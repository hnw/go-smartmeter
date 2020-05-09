package smartmeter

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/tarm/serial"
)

var (
	reVersion        = regexp.MustCompile(`(?m)^EVER\s+(.*)$`)
	reInfo           = regexp.MustCompile(`(?m)^EINFO\s+(.*)$`) // <IPADDR> + <ADDR64> + <CHANNEL> + <PANID> + <ADDR16>
	reRegisterValue  = regexp.MustCompile(`(?m)^ESREG\s+(.*)$`)
	rePanDesc        = regexp.MustCompile(`(?m)^EPANDESC$`)
	rePanChannel     = regexp.MustCompile(`(?m)^\s+Channel:([23][0-9A-F])$`)
	rePanID          = regexp.MustCompile(`(?m)^\s+Pan ID:(.*)$`)
	rePanMacAddr     = regexp.MustCompile(`(?m)^\s+Addr:(.*)$`)
	reIPAddr         = regexp.MustCompile(`(?m)^(?:[\dA-F]{4}:){7}[\dA-F]{4}$`)
	reNeibour        = regexp.MustCompile(`(?m)^((?:[\dA-F]{4}:){7}[\dA-F]{4}) [\dA-F]{16} FFFF$`)
	reEchonetLiteUDP = regexp.MustCompile(`(?m)^ERXUDP (?:[\dA-F]{4}:){7}[\dA-F]{4} (?:[\dA-F]{4}:){7}[\dA-F]{4} 0E1A 0E1A [\dA-F]{16} \d(?: \d+)? ([\dA-F]+) (.*)$`)
)

// Device
type Device struct {
	SerialPort  string
	ID          string
	Password    string
	Channel     string
	PanID       string
	MacAddr     string
	IPAddr      string
	DualStackSK bool
	Debug       bool
	Options     []Option
	inputChan   chan string
	writer      *bufio.Writer
}

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
		Options: opts,
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
		defer sr.Close()

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

func (d *Device) GetVersion(opts ...Option) (version string, err error) {
	res, err := d.QuerySKCommand("SKVER", opts...)
	if err != nil {
		return
	}
	matched := reVersion.FindStringSubmatch(res)
	if len(matched) == 0 {
		err = fmt.Errorf("Unexpected response for SKVER: %s", res)
	} else {
		version = matched[1]
	}
	return
}

func (d *Device) GetInfo(opts ...Option) (info string, err error) {
	res, err := d.QuerySKCommand("SKINFO", opts...)
	if err != nil {
		return
	}
	matched := reInfo.FindStringSubmatch(res)
	if len(matched) == 0 {
		err = fmt.Errorf("Unexpected response for SKINFO: %s", res)
	} else {
		info = matched[1]
	}
	return
}

func (d *Device) GetRegisterValue(regName string, opts ...Option) (registerValue string, err error) {
	if !strings.HasPrefix(regName, "S") {
		return "", fmt.Errorf("Invalid register name: %s")
	}
	res, err := d.QuerySKCommand("SKSREG "+regName, opts...)
	if err != nil {
		return
	}
	matched := reRegisterValue.FindStringSubmatch(res)
	if len(matched) == 0 {
		err = fmt.Errorf("Unexpected response for SKSREG: %s", res)
	} else {
		registerValue = matched[1]
	}
	return
}

func (d *Device) SetRegisterValue(regName string, regValue string, opts ...Option) (err error) {
	if !strings.HasPrefix(regName, "S") {
		return fmt.Errorf("Invalid register name: %s")
	}
	cmd := fmt.Sprintf("SKSREG %s %s", regName, regValue)
	_, err = d.QuerySKCommand(cmd, opts...)
	return
}

func (d *Device) SetID(opts ...Option) (err error) {
	if d.ID == "" {
		return errors.New("ID not specifed")
	}
	_, err = d.QuerySKCommand("SKSETRBID "+d.ID, opts...)
	return
}

func (d *Device) SetPassword(opts ...Option) (err error) {
	if d.Password == "" {
		return errors.New("Password not specifed")
	}
	cmd := fmt.Sprintf("SKSETPWD %X %s", len(d.Password), d.Password)
	_, err = d.QuerySKCommand(cmd, opts...)
	return
}

func (d *Device) GetNeibourIP(opts ...Option) (ipAddr string, err error) {
	res, err := d.QuerySKCommand("SKTABLE 2", opts...)
	if err != nil {
		return
	}
	matched := reNeibour.FindAllStringSubmatch(res, -1)
	if len(matched) != 1 {
		err = fmt.Errorf("Unexpected response for SKSREG: %s", res)
	} else {
		ipAddr = matched[0][1]
	}
	return
}

func (d *Device) getIPAddrFromMacAddr(opts ...Option) (ipAddr string, err error) {
	callback := func(line string) (bool, error) {
		// SKLL64コマンドだけはOKを返さず、直後の1行がレスポンス
		return true, nil
	}
	opts = append([]Option{Reader(callback)}, opts...)
	res, err := d.QuerySKCommand("SKLL64 "+d.MacAddr, opts...)
	ipAddr = reIPAddr.FindString(res)
	if ipAddr == "" {
		err = fmt.Errorf(`IP address is invalid: "%s"`, res)
	}
	return
}

func (d *Device) Scan(opts ...Option) (err error) {
	if err = d.SetID(); err != nil {
		fmt.Printf("%v", err)
		return
	}
	if err = d.SetPassword(); err != nil {
		fmt.Printf("%v", err)
		return
	}

	var mask uint32
	mask = 0xffffffff
	if d.Channel != "" {
		var i int64
		i, err = strconv.ParseInt(d.Channel, 16, 0)
		if err != nil {
			err = fmt.Errorf(`Specified channel is invalid: "%s"`, d.Channel)
			return
		} else if i < 33 || i > 60 {
			err = fmt.Errorf(`Channel must be 21-3C: "%s"`, d.Channel)
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
	opts = append([]Option{Reader(callback)}, opts...)
	res, err := d.QuerySKCommand(cmd, opts...)
	if err != nil {
		return
	}
	if !rePanDesc.MatchString(res) {
		err = fmt.Errorf(`Scan failed. Response is: "%s"`, res)
		return
	}

	channel := rePanChannel.FindStringSubmatch(res)[1]
	panID := rePanID.FindStringSubmatch(res)[1]
	macAddr := rePanMacAddr.FindStringSubmatch(res)[1]
	if channel == "" || panID == "" || macAddr == "" {
		err = fmt.Errorf(`Channel or PAN ID or MAC address is invalid: "%s", "%s", "%s"`, channel, panID, macAddr)
		return
	}
	d.Channel = channel
	d.PanID = panID
	d.MacAddr = macAddr

	ipAddr, err := d.getIPAddrFromMacAddr()
	if err != nil {
		return
	}
	d.IPAddr = ipAddr
	return
}

func (d *Device) Join(opts ...Option) (err error) {
	callback := func(line string) (bool, error) {
		if strings.HasPrefix(line, "EVENT 24 ") {
			// EVENT 24: PANAによる接続過程でエラーが発生した
			return false, fmt.Errorf("PANA connection error (%s) %w", line, RetryableError)
		} else if strings.HasPrefix(line, "EVENT 25 ") {
			// EVENT 25: PANAによる接続が完了した（Join成功）
			return true, nil
		}
		return false, nil
	}
	opts = append([]Option{Reader(callback)}, opts...)
	_, err = d.QuerySKCommand("SKJOIN "+d.IPAddr, opts...)
	return
}

func (d *Device) Authenticate(opts ...Option) (err error) {
	err = d.Scan(opts...)
	if err != nil {
		return
	}

	if err = d.SetRegisterValue("S02", d.Channel, opts...); err != nil {
		return
	}

	if err = d.SetRegisterValue("S03", d.PanID, opts...); err != nil {
		return
	}
	return d.Join(opts...)
}

func (d *Device) QuerySKCommand(cmd string, opts ...Option) (res string, err error) {
	query, err := NewSKQuery(d, cmd, append(d.Options, opts...)...)
	if err != nil {
		return
	}
	return query.Exec()
}

func (d *Device) QueryEchonetLite(req *Frame, opts ...Option) (res *Frame, err error) {
	secure := 1
	port := 3610
	side := 0 // 0: B-route, 1: HAN

	if d.IPAddr == "" {
		err = errors.New("IP address for smart electric energy meter is not specifed")
		return
	}

	rawFrame := req.Build()
	var cmd string
	if d.DualStackSK {
		cmd = fmt.Sprintf("SKSENDTO %d %s %04X %d %d %04X %s", secure, d.IPAddr, port, secure, side, len(rawFrame), rawFrame)
	} else {
		cmd = fmt.Sprintf("SKSENDTO %d %s %04X %d %04X %s", secure, d.IPAddr, port, secure, len(rawFrame), rawFrame)
	}

	callback := func(line string) (bool, error) {
		if strings.HasPrefix(line, "EVENT 21 ") {
			// EVENT 21: UDP送信完了
			if strings.HasSuffix(line, " 01") {
				// 01: UDP送信失敗
				return false, fmt.Errorf("Failed to send UDP packet (%s) %w", line, RetryableError)
			} else if strings.HasSuffix(line, " 02") {
				// 02: アドレス要請
				return false, fmt.Errorf("PANA unconnected (%s)", line)
			}
		} else if strings.HasPrefix(line, "ERXUDP ") {
			f, err := parseERXUDP(line)
			/*
				if err != nil {
					fmt.Printf("err=%v\n", err)
				}
			*/
			if err == nil && f.CorrespondTo(req) {
				res = f
				return true, nil
			}
		}
		return false, nil
	}
	opts = append([]Option{Reader(callback)}, opts...)
	_, err = d.QuerySKCommand(cmd, opts...)
	return
}

// ERXUDPイベント行を受け取ってFrameを返す
// ECHONET Liteのフレームのみ処理する
func parseERXUDP(line string) (res *Frame, err error) {
	matched := reEchonetLiteUDP.FindStringSubmatch(line)
	if len(matched) == 0 {
		err = fmt.Errorf("Unknown ERXUDP format: %s", line)
		return
	}

	dataLen, err := strconv.ParseInt(matched[1], 16, 32)
	if err != nil {
		err = errors.New("ERXUDP parse error (not a number) : " + line)
		return
	}
	data := matched[2]
	var rawData []byte
	if len(data) == int(dataLen) {
		// WOPT 0（バイナリ）
		rawData = []byte(data)
	} else if len(data) == int(2*dataLen) {
		// WOPT 1（16進ASCII）
		rawData, err = hex.DecodeString(data)
		if err != nil {
			err = errors.New("ERXUDP parse error (not a hexadecimal) : " + line)
			return
		}
	} else {
		err = errors.New("ERXUDP data length mismatch: " + line)
		return
	}
	return ParseFrame(rawData)
}
