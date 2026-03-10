// Package smartmeter provides an ECHONET Lite client for smart meters.
package smartmeter

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/tarm/serial"
)

var (
	reVersion = regexp.MustCompile(`(?m)^EVER\s+(.*)$`)
	// <IPADDR> + <ADDR64> + <CHANNEL> + <PANID> + <ADDR16>
	reInfo           = regexp.MustCompile(`(?m)^EINFO\s+(.*)$`)
	reRegisterValue  = regexp.MustCompile(`(?m)^ESREG\s+(.*)$`)
	rePanDesc        = regexp.MustCompile(`(?m)^EPANDESC$`)
	rePanChannel     = regexp.MustCompile(`(?m)^\s+Channel:([23][0-9A-F])$`)
	rePanID          = regexp.MustCompile(`(?m)^\s+Pan ID:(.*)$`)
	rePanMacAddr     = regexp.MustCompile(`(?m)^\s+Addr:(.*)$`)
	reIPAddr         = regexp.MustCompile(`(?m)^(?:[\dA-F]{4}:){7}[\dA-F]{4}$`)
	reNeibour        = regexp.MustCompile(`(?m)^((?:[\dA-F]{4}:){7}[\dA-F]{4}) [\dA-F]{16} FFFF$`)
	reEchonetLiteUDP = regexp.MustCompile(
		`(?m)^ERXUDP (?:[\dA-F]{4}:){7}[\dA-F]{4} ` +
			`(?:[\dA-F]{4}:){7}[\dA-F]{4} ([\dA-F]{4}) ([\dA-F]{4}) [\dA-F]{16} ` +
			`\d(?: \d+)? ([\dA-F]+) (.*)$`,
	)
	errNonEchonetLiteERXUDP = errors.New("non-echonet ERXUDP")
)

// Device represents a smart meter connection and its settings.
type Device struct {
	SerialPort  string
	ID          string
	Password    string
	Channel     string
	IPAddr      string
	DualStackSK bool
	Verbosity   int

	panID     string
	macAddr   string
	logger    *log.Logger
	options   []Option
	inputChan chan string
	writer    *bufio.Writer
}

// Open opens the serial port and returns a Device configured with opts.
func Open(path string, opts ...Option) (d *Device, err error) {
	c := &serial.Config{
		Name:     path,
		Baud:     115200,
		Size:     8,
		StopBits: 1,
	}
	sr, err := serial.OpenPort(c)
	if err != nil {
		return
	}

	d = &Device{
		options: opts,
		writer:  bufio.NewWriter(sr),
	}
	for _, opt := range opts {
		if err := opt(d); err != nil {
			return nil, err
		}
	}

	scanner := bufio.NewScanner(sr)
	ch := make(chan string, 4)
	d.inputChan = ch

	go func() {
		defer close(ch)
		defer func() {
			if err := sr.Close(); err != nil {
				log.Printf("smartmeter: failed to close serial port: %v", err)
			}
		}()

		for scanner.Scan() {
			line := scanner.Text()
			ch <- line
		}
		/*
			if err := scanner.Err(); err != nil {
				log.Fatal(err)
			}
		*/
	}()

	return
}

// GetVersion returns the version string from SKVER.
func (d *Device) GetVersion(opts ...Option) (version string, err error) {
	res, err := d.QuerySKCommand("SKVER", opts...)
	if err != nil {
		return
	}
	matched := reVersion.FindStringSubmatch(res)
	if len(matched) == 0 {
		err = fmt.Errorf("unexpected response for SKVER: %s", res)
	} else {
		version = matched[1]
	}
	return
}

// GetInfo returns the info string from SKINFO.
func (d *Device) GetInfo(opts ...Option) (info string, err error) {
	res, err := d.QuerySKCommand("SKINFO", opts...)
	if err != nil {
		return
	}
	matched := reInfo.FindStringSubmatch(res)
	if len(matched) == 0 {
		err = fmt.Errorf("unexpected response for SKINFO: %s", res)
	} else {
		info = matched[1]
	}
	return
}

// GetRegisterValue returns a register value from SKSREG.
func (d *Device) GetRegisterValue(
	regName string,
	opts ...Option,
) (registerValue string, err error) {
	if !strings.HasPrefix(regName, "S") {
		return "", fmt.Errorf("invalid register name: %s", regName)
	}
	res, err := d.QuerySKCommand("SKSREG "+regName, opts...)
	if err != nil {
		return
	}
	matched := reRegisterValue.FindStringSubmatch(res)
	if len(matched) == 0 {
		err = fmt.Errorf("unexpected response for SKSREG: %s", res)
	} else {
		registerValue = matched[1]
	}
	return
}

// SetRegisterValue sets a register value via SKSREG.
func (d *Device) SetRegisterValue(regName string, regValue string, opts ...Option) (err error) {
	if !strings.HasPrefix(regName, "S") {
		return fmt.Errorf("invalid register name: %s", regName)
	}
	cmd := fmt.Sprintf("SKSREG %s %s", regName, regValue)
	_, err = d.QuerySKCommand(cmd, opts...)
	return
}

// SetID sets the B-route authentication ID on the device.
func (d *Device) SetID(opts ...Option) (err error) {
	if d.ID == "" {
		return errors.New("id not specified")
	}
	_, err = d.QuerySKCommand("SKSETRBID "+d.ID, opts...)
	return
}

// SetPassword sets the B-route authentication password on the device.
func (d *Device) SetPassword(opts ...Option) (err error) {
	if d.Password == "" {
		return errors.New("password not specified")
	}
	cmd := fmt.Sprintf("SKSETPWD %X %s", len(d.Password), d.Password)
	_, err = d.QuerySKCommand(cmd, opts...)
	return
}

// GetNeibourIP returns the neighbor IP address from SKTABLE 2.
func (d *Device) GetNeibourIP(opts ...Option) (ipAddr string, err error) {
	res, err := d.QuerySKCommand("SKTABLE 2", opts...)
	if err != nil {
		return
	}
	matched := reNeibour.FindAllStringSubmatch(res, -1)
	if len(matched) != 1 {
		err = fmt.Errorf("unexpected response for SKTABLE 2: %s", res)
	} else {
		ipAddr = matched[0][1]
	}
	return
}

func (d *Device) getIPAddrFromMacAddr(opts ...Option) (ipAddr string, err error) {
	callback := func(_ string) (bool, error) {
		// SKLL64コマンドだけはOKを返さず、直後の1行がレスポンス
		return true, nil
	}
	skll64Opts := append([]Option{Reader(callback)}, opts...)
	res, err := d.QuerySKCommand("SKLL64 "+d.macAddr, skll64Opts...)
	ipAddr = reIPAddr.FindString(res)
	if ipAddr == "" {
		err = fmt.Errorf(`ip address is invalid: %q`, res)
	}
	return
}

// Scan performs an active scan and populates channel, PAN ID, MAC address, and IP address.
func (d *Device) Scan(opts ...Option) (err error) {
	if err = d.SetID(); err != nil {
		return
	}
	if err = d.SetPassword(); err != nil {
		return
	}

	var mask uint32
	mask = 0xffffffff
	if d.Channel != "" {
		var i int64
		i, err = strconv.ParseInt(d.Channel, 16, 0)
		if err != nil {
			err = fmt.Errorf(`specified channel is invalid: "%s"`, d.Channel)
			return
		} else if i < 33 || i > 60 {
			err = fmt.Errorf(`channel must be 21-3C: "%s"`, d.Channel)
			return
		}
		mask = 1 << (i - 33)
	}
	cmd := fmt.Sprintf("SKSCAN 2 %08X 7", mask)
	if d.DualStackSK {
		cmd = cmd + " 0"
	}

	callback := func(line string) (bool, error) {
		if strings.HasPrefix(line, "EVENT 22 ") {
			// EVENT 22: アクティブスキャン完了
			return true, nil
		}
		return false, nil
	}
	skscanOpts := append([]Option{Reader(callback)}, opts...)
	res, err := d.QuerySKCommand(cmd, skscanOpts...)
	if err != nil {
		return
	}
	if !rePanDesc.MatchString(res) {
		err = fmt.Errorf(`scan failed. response is: "%s"`, res)
		return
	}

	channel := rePanChannel.FindStringSubmatch(res)[1]
	panID := rePanID.FindStringSubmatch(res)[1]
	macAddr := rePanMacAddr.FindStringSubmatch(res)[1]
	if channel == "" || panID == "" || macAddr == "" {
		err = fmt.Errorf(
			`channel or PAN ID or MAC address is invalid: "%s", "%s", "%s"`,
			channel,
			panID,
			macAddr,
		)
		return
	}
	d.Channel = channel
	d.panID = panID
	d.macAddr = macAddr

	ipAddr, err := d.getIPAddrFromMacAddr(opts...)
	if err != nil {
		return
	}
	d.IPAddr = ipAddr
	return
}

// Join connects to the meter using SKJOIN.
func (d *Device) Join(opts ...Option) (err error) {
	callback := func(line string) (bool, error) {
		if strings.HasPrefix(line, "EVENT 24 ") {
			// EVENT 24: PANAによる接続過程でエラーが発生した
			return false, fmt.Errorf("pana connection error (%s). %w", line, ErrRetryable)
		} else if strings.HasPrefix(line, "EVENT 25 ") {
			// EVENT 25: PANAによる接続が完了した（Join成功）
			return true, nil
		}
		return false, nil
	}
	joinOpts := append([]Option{Reader(callback)}, opts...)
	_, err = d.QuerySKCommand("SKJOIN "+d.IPAddr, joinOpts...)
	return
}

// Authenticate performs scan, register configuration, and join.
func (d *Device) Authenticate(opts ...Option) (err error) {
	err = d.Scan(opts...)
	if err != nil {
		return
	}

	if err = d.SetRegisterValue("S02", d.Channel, opts...); err != nil {
		return
	}

	if err = d.SetRegisterValue("S03", d.panID, opts...); err != nil {
		return
	}
	return d.Join(opts...)
}

// QuerySKCommand sends an SK command and returns the response text.
func (d *Device) QuerySKCommand(cmd string, opts ...Option) (res string, err error) {
	query, err := newSKQuery(d, cmd, append(d.options, opts...)...)
	if err != nil {
		d.warnf("Error for SK command %q: %+v", cmd, err)
		return
	}
	res, err = query.Exec()
	if err != nil {
		d.warnf("Error for SK command %q: %+v", cmd, err)
	}
	return
}

// QueryEchonetLite sends an ECHONET Lite request and waits for the response.
func (d *Device) QueryEchonetLite(req *Frame, opts ...Option) (res *Frame, err error) {
	secure := 1
	port := 3610
	side := 0 // 0: B-route, 1: HAN

	if d.IPAddr == "" {
		err = errors.New("ip address for smart electric energy meter is not specified")
		return
	}

	rawFrame := req.Build()
	var cmd string
	if d.DualStackSK {
		cmd = fmt.Sprintf(
			"SKSENDTO %d %s %04X %d %d %04X %s",
			secure,
			d.IPAddr,
			port,
			secure,
			side,
			len(rawFrame),
			rawFrame,
		)
	} else {
		cmd = fmt.Sprintf(
			"SKSENDTO %d %s %04X %d %04X %s",
			secure,
			d.IPAddr,
			port,
			secure,
			len(rawFrame),
			rawFrame,
		)
	}

	callback := func(line string) (bool, error) {
		if strings.HasPrefix(line, "EVENT 21 ") {
			// EVENT 21: UDP送信完了
			if strings.HasSuffix(line, " 01") {
				// 01: UDP送信失敗
				return false, fmt.Errorf(
					"failed to send UDP packet (EVENT 21/01). %w",
					ErrRetryable,
				)
			} else if strings.HasSuffix(line, " 02") {
				// 02: アドレス要請
				return false, fmt.Errorf("pana unconnected (EVENT 21/02)")
			}
		} else if strings.HasPrefix(line, "ERXUDP ") {
			frame, parseErr := parseERXUDP(line)
			if parseErr != nil {
				if errors.Is(parseErr, errNonEchonetLiteERXUDP) {
					d.debugf("Ignored non-echonet ERXUDP: %s", line)
					return false, nil
				}
				d.warnf("ERXUDP parse error: cmd=%q, err=%+v", cmd, parseErr)
			} else if !frame.CorrespondTo(req) {
				d.debugf("ERXUDP ignorable error: f=%+v, req=%+v", frame, req)
			} else {
				res = frame
				return true, nil
			}
		}
		return false, nil
	}
	echonetLiteOpts := append([]Option{Reader(callback)}, opts...)
	_, err = d.QuerySKCommand(cmd, echonetLiteOpts...)
	return
}

// ERXUDPイベント行を受け取ってFrameを返す
// ECHONET Liteのフレームのみ処理する（それ以外は errNonEchonetLiteERXUDP を返す）
func parseERXUDP(line string) (*Frame, error) {
	matched := reEchonetLiteUDP.FindStringSubmatch(line)
	if len(matched) == 0 {
		return nil, fmt.Errorf("unknown ERXUDP format: %s", line)
	}

	srcPort := matched[1]
	dstPort := matched[2]
	if !strings.EqualFold(srcPort, "0E1A") || !strings.EqualFold(dstPort, "0E1A") {
		return nil, errNonEchonetLiteERXUDP
	}

	dataLen, err := strconv.ParseInt(matched[3], 16, 32)
	if err != nil {
		return nil, fmt.Errorf("ERXUDP parse error (not a number) : %s", line)
	}
	data := matched[4]
	var rawData []byte
	if len(data) == int(dataLen) {
		// WOPT 0（バイナリ）
		rawData = []byte(data)
	} else if len(data) == int(2*dataLen) {
		// WOPT 1（16進ASCII）
		rawData, err = hex.DecodeString(data)
		if err != nil {
			return nil, fmt.Errorf("ERXUDP parse error (not a hexadecimal) : %s", line)
		}
	} else {
		return nil, fmt.Errorf("ERXUDP data length mismatch: %s", line)
	}
	return ParseFrame(rawData)
}

func (d *Device) warnf(fmt string, v ...interface{}) {
	if d.Verbosity >= 1 {
		d.logf(fmt, v...)
	}
}

func (d *Device) infof(fmt string, v ...interface{}) {
	if d.Verbosity >= 2 {
		d.logf(fmt, v...)
	}
}

func (d *Device) debugf(fmt string, v ...interface{}) {
	if d.Verbosity >= 3 {
		d.logf(fmt, v...)
	}
}

func (d *Device) logf(fmt string, v ...interface{}) {
	if d.logger != nil {
		d.logger.Printf(fmt, v...)
	} else {
		log.Printf(fmt, v...)
	}
}
