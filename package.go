package pack

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/zhangpeihao/log"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

const (
	PACKAGE_HEAD_TAIL   uint16 = 0X1F9E
	PACKAGE_HEAD_TAIL_1 uint8  = 0X9E
	PACKAGE_HEAD_TAIL_2 uint8  = 0X1F
	PACKAGE_HEAD_LEN    uint16 = 6
	PACKAGE_MAX_LENGTH  int    = 65535
	PACKAGE_LOG_NAME           = "pack"
)

const (
	PACKAGE_MEMBER_TYPE_A = byte('A')
	PACKAGE_MEMBER_TYPE_B = byte('B')
	PACKAGE_MEMBER_TYPE_C = byte('C')
)

var (
	package_logger     *log.Logger
	package_logHandler log.LoggerModule
)

type PackageMember *[]byte
type PackageMembers map[byte]PackageMember

type Package struct {
	MsgId    uint16
	dataSize uint16
	members  *PackageMembers
}

func InitPackageLog(logger *log.Logger) {
	package_logger = logger
	package_logHandler = logger.LoggerModule(PACKAGE_LOG_NAME)
}

func (pack *Package) Members() *PackageMembers {
	return pack.members
}

func (pack *Package) Member(memberType byte) (member PackageMember) {
	member, _ = (*pack.members)[memberType]
	return
}

func (pack *Package) SetMembers(members *PackageMembers) {
	pack.members = members
}

func (pack *Package) AddMember(memberType byte, member PackageMember) {
	(*pack.members)[memberType] = member
}

func (pack *Package) AddStringMember(memberType byte, str string) {
	member := []byte(str)
	(*pack.members)[memberType] = &member
}

func (pack *Package) AMember() (member PackageMember, err error) {
	var found bool
	member, found = (*pack.members)[PACKAGE_MEMBER_TYPE_A]
	if !found {
		return nil, errors.New("member A is nil")
	}
	return
}

func (pack *Package) BMember() (member PackageMember, err error) {
	var found bool
	member, found = (*pack.members)[PACKAGE_MEMBER_TYPE_B]
	if !found {
		return nil, errors.New("member B is nil")
	}
	return
}

func (pack *Package) CMember() (member PackageMember, err error) {
	var found bool
	member, found = (*pack.members)[PACKAGE_MEMBER_TYPE_C]
	if !found {
		return nil, errors.New("member C is nil")
	}
	return
}

func (pack *Package) TwoMembers() (memberA PackageMember, memberB PackageMember, err error) {
	var found bool
	memberA, found = (*pack.members)[PACKAGE_MEMBER_TYPE_A]
	if !found {
		return nil, nil, errors.New("member A is nil")
	}
	memberB, found = (*pack.members)[PACKAGE_MEMBER_TYPE_B]
	if !found {
		return nil, nil, errors.New("member B is nil")
	}
	return
}

func (pack *Package) ThreeMembers() (memberA PackageMember, memberB PackageMember, memberC PackageMember, err error) {
	var found bool
	memberA, found = (*pack.members)[PACKAGE_MEMBER_TYPE_A]
	if !found {
		return nil, nil, nil, errors.New("member A is nil")
	}
	memberB, found = (*pack.members)[PACKAGE_MEMBER_TYPE_B]
	if !found {
		return nil, nil, nil, errors.New("member B is nil")
	}
	memberC, found = (*pack.members)[PACKAGE_MEMBER_TYPE_C]
	if !found {
		return nil, nil, nil, errors.New("member C is nil")
	}
	return
}

func (pack *Package) Dump() {
	if package_logger.ModuleLevelCheck(package_logHandler, log.LOG_LEVEL_DEBUG) {
		package_logger.ForcePrintf("Package(MsgID: %d)\n", pack.MsgId)
		if pack.members != nil && len(*pack.members) > 0 {
			for k, member := range *pack.members {
				package_logger.ForcePrintf("\t%c: %s\n", k, string(*member))
			}
		} else {
			package_logger.ForcePrintf("\tEMPTY\n")
		}
	}
}

func (pack *Package) DumpStdout() {
	fmt.Printf("Package(MsgID: %d)\n", pack.MsgId)
	if pack.members != nil && len(*pack.members) > 0 {
		for k, member := range *pack.members {
			fmt.Printf("\t%c: %s\n", k, string(*member))
		}
	} else {
		fmt.Printf("\tEMPTY\n")
	}
}

/////////////////////////////////////////////////////////////
// Parse from stream
func ParsePackage(reader io.Reader) (pack *Package, err error) {
	head := make([]byte, PACKAGE_HEAD_LEN)
	_, err = io.ReadAtLeast(reader, head, int(PACKAGE_HEAD_LEN))
	if err != nil {
		return nil, err
	}
	pack, err = parsePackageHead(head)
	if err != nil {
		return nil, err
	}

	data := make([]byte, pack.dataSize)
	_, err = io.ReadAtLeast(reader, data, int(pack.dataSize))
	if err != nil {
		return nil, err
	}
	pack.members, err = parseMembers(&data)
	if err != nil {
		return nil, err
	}
	return
}

func ParsePackageForHtml(msgId uint16, resp *http.Response) (pack *Package, err error) {
	var data []byte
	var dataSize int
	data, err = ioutil.ReadAll(resp.Body)
	dataSize = len(data)

	pack = &Package{
		MsgId:    msgId,
		dataSize: uint16(dataSize),
	}

	pack.members, err = parseMembers(&data)
	if err != nil {
		return nil, err
	}
	return
}

func ParsePackageForHtmlRequest(req *http.Request) (pack *Package, err error) {
	// Get last part of URI as message ID
	if len(req.RequestURI) == 0 {
		package_logger.ModulePrintf(package_logHandler, log.LOG_LEVEL_TRACE, "Request URI is empty! from %s", req.RemoteAddr)
		return nil, errors.New("Request URI is empty")
	}
	parts := strings.Split(req.RequestURI, "/")
	function := parts[len(parts)-1]
	var msgId int
	if msgId, err = strconv.Atoi(function); err != nil {
		return nil, err
	}

	pack = &Package{
		MsgId: uint16(msgId),
	}
	members := make(PackageMembers)
	pack.members = &members

	if err = req.ParseForm(); err != nil {
		package_logger.ModulePrintf(package_logHandler, log.LOG_LEVEL_TRACE, "Parse form err: %s! from %s", err.Error(), req.RemoteAddr)
		return nil, err
	}
	for key, value := range req.Form {
		if len(key) != 1 {
			package_logger.ModulePrintf(package_logHandler, log.LOG_LEVEL_TRACE, "Member key %s from %s\n", key, req.RemoteAddr)
			continue
		}
		valueLen := len(value)
		if valueLen == 0 {
			package_logger.ModulePrintf(package_logHandler, log.LOG_LEVEL_TRACE, "Member value is empty from %s\n", key, req.RemoteAddr)
			continue
		}
		valueBytes := []byte(value[0])
		pack.AddMember([]byte(key)[0], &valueBytes)
		pack.dataSize += uint16(3 + len(valueBytes))
	}

	return
}

func parsePackageHead(data []byte) (pack *Package, err error) {
	if data[4] != PACKAGE_HEAD_TAIL_1 || data[5] != PACKAGE_HEAD_TAIL_2 {
		return nil, errors.New(fmt.Sprintf("Head error! data: % 0x", data))
	}
	pack = new(Package)
	buf := bytes.NewBuffer(data)
	err = binary.Read(buf, binary.LittleEndian, &(pack.MsgId))
	if err != nil {
		return nil, err
	}
	err = binary.Read(buf, binary.LittleEndian, &(pack.dataSize))
	if err != nil {
		return nil, err
	}
	return
}

func parseMembers(data *[]byte) (*PackageMembers, error) {
	var err error
	buf := bytes.NewBuffer(*data)
	var datalen uint16
	var memberType byte
	members := make(PackageMembers)
	for buf.Len() > 0 {
		memberType, err = buf.ReadByte()
		if err != nil {
			return nil, err
		}
		//    fmt.Printf("memberType: %s\n", string(memberType))
		err = binary.Read(buf, binary.LittleEndian, &datalen)
		if err != nil {
			return nil, err
		}
		//    fmt.Printf("memberLen: %d\n", datalen)
		if int(datalen) > buf.Len() {
			return nil, errors.New("EOF")
		}
		member := buf.Next(int(datalen))
		//    fmt.Printf("member: %s\n", string(member))
		members[memberType] = &member
	}
	return &members, nil
}

/////////////////////////////////////////////////////////////////
// Create new package
func NewPackage(msgId uint16, members *PackageMembers) (pack *Package) {
	pack = new(Package)
	pack.MsgId = msgId
	if members == nil {
		ms := make(PackageMembers)
		members = &ms
	}
	pack.members = members
	return
}

func NewPackageWithString(msgId uint16, memberType byte, str *string) (pack *Package) {
	member := []byte(*str)
	members := make(PackageMembers)
	members[memberType] = &member
	return NewPackage(msgId, &members)
}

func NewPackageWithData(msgId uint16, memberType byte, data *[]byte) (pack *Package) {
	members := make(PackageMembers)
	members[memberType] = data
	return NewPackage(msgId, &members)
}

/////////////////////////////////////////////////////////////////
// Write to stream
func (pack *Package) Write(writer io.Writer) (n int, err error) {
	var tn int
	var headData []byte
	if pack.members == nil {
		pack.dataSize = 0
	} else {
		dataSize := 0
		for _, member := range *pack.members {
			if member != nil {
				dataSize = dataSize + 3 + len(*member)
			}
		}
		if dataSize > PACKAGE_MAX_LENGTH {
			// Todo: switch to large package mode
			return 0, errors.New(fmt.Sprintf("Packet size overflow, datasize: %d", dataSize))
		}
		pack.dataSize = uint16(dataSize)
	}
	headData, err = pack.headBytes()
	if err != nil {
		return
	}
	tn, err = writer.Write(headData)
	//  fmt.Printf("len(headData): %d, tn: %d\n", len(headData), tn)
	if err != nil {
		return
	}
	n = tn
	for memberType, member := range *pack.members {
		writer.Write([]byte{memberType})
		n++
		buf := new(bytes.Buffer)
		memberLen := 0
		if member != nil {
			memberLen = len(*member)
		}
		err = binary.Write(buf, binary.LittleEndian, uint16(memberLen))
		if err != nil {
			return
		}
		tn, err = writer.Write(buf.Bytes())
		if err != nil {
			return
		}
		if tn != 2 {
			return n + tn, errors.New("Member len value is not 16bit: " + strconv.Itoa(tn))
		}
		n = n + tn
		if memberLen > 0 {
			tn, err = writer.Write(*member)
			if err != nil {
				return
			}
			n = n + tn
		}
	}
	return
}

func (pack *Package) headBytes() (headData []byte, err error) {
	buf := new(bytes.Buffer)
	err = binary.Write(buf, binary.LittleEndian, pack.MsgId)
	if err != nil {
		return nil, err
	}
	err = binary.Write(buf, binary.LittleEndian, pack.dataSize)
	if err != nil {
		return nil, err
	}
	err = binary.Write(buf, binary.LittleEndian, PACKAGE_HEAD_TAIL)
	if err != nil {
		return nil, err
	}
	headData = buf.Bytes()
	return
}
