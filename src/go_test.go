// test transporter
package ftunnel

import (
	"crypto/md5"
	"errors"
	"fmt"
	flatbuffers "github.com/google/flatbuffers/go"
	"log"
	"math/rand"
	"net"
	"testing"
	"time"
)

const (
	TEST_PORT = "63137"
)

// test transporter ReadNextPacket
func TestReadNextPacket(t *testing.T) {
	ln, err := net.Listen("tcp", "localhost:"+TEST_PORT)
	if err != nil {
		t.Fatal(err)
	}
	exit := make(chan bool, 1)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				t.Fatal(err)
			}

			tr := NewTransporter(conn)
			pack := tr.ReadNextPacket()
			if pack == nil {
				t.Fatal(errors.New("ReadNextPacket return nil (0)"))
			}
			if pack.Command() != CMD_PING {
				log.Println("pack.Command()", pack.Command(), CMD_PING)
				t.Fatal(errors.New("Command mismatch"))
			}

			pack = tr.ReadNextPacket()
			if pack == nil {
				t.Fatal(errors.New("ReadNextPacket return nil (1)"))
			}
			if pack.Command() != CMD_PONG {
				log.Println("pack.Command()", pack.Command(), CMD_PONG)
				t.Fatal(errors.New("Command mismatch"))
			}

			pack = tr.ReadNextPacket()
			if pack != nil {
				t.Fatal(errors.New("ReadNextPacket return non-nil (3)"))
			}

			pack = tr.ReadNextPacket()
			if pack == nil {
				t.Fatal(errors.New("ReadNextPacket return nil (2)"))
			}
			if pack.Command() != CMD_QUERY_IDENTITY {
				log.Println("pack.Command()", pack.Command(), CMD_QUERY_IDENTITY)
				t.Fatal(errors.New("Command mismatch"))
			}

			pack = tr.ReadNextPacket()
			if pack == nil {
				t.Fatal(errors.New("ReadNextPacket return nil (2)"))
			}
			b := pack.ContentData()
			if len(b) < MTU || fmt.Sprintf("%X", md5.Sum(b)) != pack.DstNetwork() {
				t.Fatal(errors.New("ContentData mismatch"))
			}

			exit <- true
			break
		}
	}()

	conn, err := net.Dial("tcp", "localhost:"+TEST_PORT)
	if err != nil {
		t.Fatal(err)
	}

	tr := NewTransporter(conn)
	// Testing CMD_PING
	tr.WritePacketBytes(BuildCommandPacket(CMD_PING))

	tr.WritePacketBytes(BuildCommandPacket(CMD_PONG))

	time.Sleep(time.Second)

	// Testing unformatted bytes
	tr.WritePacketBytes([]byte("12"))

	tr.WritePacketBytes(BuildCommandPacket(CMD_QUERY_IDENTITY))

	tc := make([]byte, 3*MTU)
	for i := range tc {
		tc[i] = byte(rand.Intn(1))
	}

	builder := flatbuffers.NewBuilder(0)
	PacketStart(builder)
	PacketAddCommand(builder, CMD_CONN)
	PacketAddDstNetwork(builder, builder.CreateString(fmt.Sprintf("%X", md5.Sum(tc))))
	PacketAddContentData(builder, tc)
	builder.Finish(PacketEnd(builder))
	tr.WritePacketBytes(builder.Bytes[builder.Head():])

	tr.Close()

	select {
	case <-exit:
		fmt.Println("TestReadNextPacket done")
	case <-time.After(3 * time.Second):
		t.Fatal(errors.New("timed out"))
	}
}

// TODO: test transporter ping pong
