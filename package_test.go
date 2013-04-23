package pack

import (
	"bytes"
	"fmt"
	"testing"
)

func TestHeader(t *testing.T) {
	fmt.Println(`//////////////////// TestHeader \\\\\\\\\\\\\\\\\\\\`)
	testHeadData := []byte{0x11, 0x22, 0x33, 0x44, PACKAGE_HEAD_TAIL_1, PACKAGE_HEAD_TAIL_2}
	testMsgId := uint16(0x2211)
	testDataSize := uint16(0x4433)
	p := &Package{
		MsgId:    testMsgId,
		dataSize: testDataSize,
	}
	headData, err := p.headBytes()
	if err != nil {
		t.Error("Get head bytes error:", err)
	} else {
		if len(headData) != int(PACKAGE_HEAD_LEN) {
			t.Errorf("Got head length: %d, exprct %d\n", len(headData), PACKAGE_HEAD_LEN)
		} else {
			if headData[0] != 0x11 || headData[1] != 0x22 ||
				headData[2] != 0x33 || headData[3] != 0x44 ||
				headData[4] != PACKAGE_HEAD_TAIL_1 ||
				headData[5] != PACKAGE_HEAD_TAIL_2 {
				t.Errorf("Got head data: % 2X, expect: % 2X\n", headData, testHeadData)
			}
		}
	}

	pp, err := parsePackageHead(testHeadData)
	if err != nil {
		t.Error("parsePackageHead error:", err)
	} else {
		if pp.MsgId != testMsgId ||
			pp.dataSize != testDataSize {
			t.Errorf("parsePackageHead got: %+v, expect: %+v\n", pp, p)
		}
	}
	fmt.Println(`\\\\\\\\\\\\\\\\\\\\ TestHeader ////////////////////`)
}

func CheckPackage(t *testing.T, got *Package, expect *Package) {
	ok := false
	defer func() {
		if !ok {
			t.Errorf("Got %+v, Expect: %+v\n", got, expect)
		}
	}()

	if got.MsgId != expect.MsgId ||
		got.dataSize != expect.dataSize {
		return
	}
	gotMembers := got.Members()
	expectMembers := expect.Members()
	if len(*gotMembers) != len(*expectMembers) {
		return
	}

	for key, value := range *gotMembers {
		member := expect.Member(key)
		if member == nil {
			return
		}
		if bytes.Compare(*member, *value) != 0 {
			return
		}
	}
	ok = true
}

func TestPackage(t *testing.T) {
	fmt.Println(`//////////////////// TestPackage \\\\\\\\\\\\\\\\\\\\`)
	testMemberAString := "AString"
	testMemberBString := "BString"
	testMemberCString := "CString"
	testMemberA := []byte(testMemberAString)
	testMemberB := []byte(testMemberBString)
	testMemberC := []byte(testMemberCString)
	testMembers := make(PackageMembers)
	testMembers['A'] = &testMemberA
	testMembers['B'] = &testMemberB
	testMembers['C'] = &testMemberC

	testMsgId := uint16(0x2211)
	testDataSize := uint16(len(testMemberA) + len(testMemberB) + len(testMemberC) + 9)
	p := &Package{
		MsgId:    testMsgId,
		dataSize: uint16(0), // Data size is set when parse or compose.
		members:  &testMembers,
	}

	if &testMembers != p.Members() {
		t.Errorf("p.Members got: %+v, expect: %+v\n", p.Members(), &testMembers)
	}

	if p.Member('A') != &testMemberA {
		t.Errorf("p.Member[A]: %+v, expect: %+v\n", p.Member('A'), &testMemberA)
	}
	if p.Member('B') != &testMemberB {
		t.Errorf("p.Member[B]: %+v, expect: %+v\n", p.Member('B'), &testMemberB)
	}
	if p.Member('C') != &testMemberC {
		t.Errorf("p.Member[C]: %+v, expect: %+v\n", p.Member('C'), &testMemberC)
	}

	var member PackageMember
	var err error
	member, err = p.AMember()
	if err != nil {
		t.Error("Get p.AMember error", err)
	}
	if member != &testMemberA {
		t.Errorf("p.AMember: %+v, expect: %+v\n", member, &testMemberA)
	}
	member, err = p.BMember()
	if err != nil {
		t.Error("Get p.BMember error", err)
	}
	if member != &testMemberB {
		t.Errorf("p.BMember: %+v, expect: %+v\n", member, &testMemberB)
	}
	member, err = p.CMember()
	if err != nil {
		t.Error("Get p.CMember error", err)
	}
	if member != &testMemberC {
		t.Errorf("p.CMember: %+v, expect: %+v\n", member, &testMemberC)
	}

	var member1 PackageMember
	var member2 PackageMember
	var member3 PackageMember
	member1, member2, err = p.TwoMembers()
	if err != nil {
		t.Error("Get p.TwoMembers() error", err)
	}
	if member1 != &testMemberA {
		t.Errorf("p.AMember: %+v, expect: %+v\n", member1, &testMemberA)
	}
	if member2 != &testMemberB {
		t.Errorf("p.AMember: %+v, expect: %+v\n", member2, &testMemberB)
	}

	member1, member2, member3, err = p.ThreeMembers()
	if err != nil {
		t.Error("Get p.ThreeMembers() error", err)
	}
	if member1 != &testMemberA {
		t.Errorf("p.AMember: %+v, expect: %+v\n", member1, &testMemberA)
	}
	if member2 != &testMemberB {
		t.Errorf("p.AMember: %+v, expect: %+v\n", member2, &testMemberB)
	}
	if member3 != &testMemberC {
		t.Errorf("p.AMember: %+v, expect: %+v\n", member3, &testMemberC)
	}

	p.DumpStdout()

	p1 := NewPackage(testMsgId, &testMembers)
	CheckPackage(t, p1, p)

	buf := new(bytes.Buffer)
	var n int
	n, err = p.Write(buf)
	if err != nil {
		t.Error("p.Write() err:", err)
	}
	if n != int(PACKAGE_HEAD_LEN+testDataSize) {
		t.Errorf("p.Write() got length: %d, expect: %d\n", n, PACKAGE_HEAD_LEN+testDataSize)
	}
	if buf.Len() != int(PACKAGE_HEAD_LEN+testDataSize) {
		t.Errorf("buf length: %d, expect: %d\n", buf.Len(), PACKAGE_HEAD_LEN+testDataSize)
	}

	p2, err := ParsePackage(buf)
	if err != nil {
		t.Error("ParsePackage(buf) err:", err)
	}
	p.dataSize = testDataSize
	CheckPackage(t, p2, p)
	fmt.Println(`\\\\\\\\\\\\\\\\\\\\ TestPackage ////////////////////`)
}
