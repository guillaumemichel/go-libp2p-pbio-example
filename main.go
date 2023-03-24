package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"main/pb"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-msgio/pbio"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var ProtocolID = protocol.ID("libp2p/mymessage/1.0.0")

// SendRequest sends a pb.MyMessage to a peer and waits for a response
func SendRequest(h host.Host, p peer.ID, msg *pb.MyMessage) (*pb.MyMessage, error) {
	// open a stream to the peer
	s, err := h.NewStream(context.Background(), p, ProtocolID)
	if err != nil {
		return nil, err
	}
	defer s.Close()

	// send the message
	err = WriteMsg(s, msg)
	if err != nil {
		return nil, err
	}

	resp := &pb.MyMessage{}
	// wait for and read the response
	err = ReadMsg(s, resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// WriteMsg is a helper function that uses a delimited writer to write a
// protobuf message to a stream
func WriteMsg(s network.Stream, msg protoreflect.ProtoMessage) error {
	w := pbio.NewDelimitedWriter(s)
	return w.WriteMsg(msg)
}

// ReadMsg is a helper function that uses a delimited reader to read a
// protobuf message from a stream
func ReadMsg(s network.Stream, msg protoreflect.ProtoMessage) error {
	r := pbio.NewDelimitedReader(s, network.MessageSizeMax)
	return r.ReadMsg(msg)
}

func handleStream(s network.Stream) {
	defer s.Close()

	// create a protobuf reader and writer
	r := pbio.NewDelimitedReader(s, network.MessageSizeMax)
	w := pbio.NewDelimitedWriter(s)

	for {
		req := &pb.MyMessage{}
		// read a message from the stream
		err := r.ReadMsg(req)
		if err != nil {
			if err == io.EOF {
				// stream EOF, all done
				return
			}
			fmt.Println(err)
			return
		}
		fmt.Println("Server got request:", req.Field)
		resp := &pb.MyMessage{
			Field: "Pong",
		}
		// write the response to the stream
		err = w.WriteMsg(resp)
		if err != nil {
			return
		}
	}
}

func main() {
	ctx := context.Background()

	// Set up a libp2p server
	server, err := libp2p.New()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Set up a a libp2p client
	client, err := libp2p.New()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// set a stream handler on server
	server.SetStreamHandler(ProtocolID, handleStream)

	// connect the client to the server
	err = client.Connect(ctx, server.Peerstore().PeerInfo(server.ID()))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// define protobuf message
	req := &pb.MyMessage{
		Field: "Ping",
	}
	// send request to server, wait for response
	resp, err := SendRequest(client, server.ID(), req)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("Client got response:", resp.Field)
}
