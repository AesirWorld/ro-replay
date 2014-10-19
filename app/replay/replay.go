package replay

import (
	//	"encoding/json"
	"encoding/json"
	"fmt"
	"time"

	zmq "github.com/pebbe/zmq4"
)

// Config
type Replay struct {
}

// Json auth check request
type AuthCheck struct {
	AccountID int `json:"account_id"`
	AuthCode  int `json:"auth_code"`
}

// Json auth check response
type AuthResp struct {
	Auth bool `json:"auth"`
}

// Json broadcast request stream operation
type StreamReq struct {
	AccountID int `json:"account_id"`
}

func (r *Replay) Register(account_id int, auth_code int) (chan string, error) {
	/*
		auth, err := r.Auth(account_id, auth_code)

		if err != nil {
			return nil, err
		}

		if auth == false {
			return nil, ErrAuthUnauthorized
		}
	*/

	// Broadcast request to start streaming
	publisher, err := zmq.NewSocket(zmq.PUB)
	if err != nil {
		panic(err)
	}
	publisher.SetIdentity("ro-replay")
	publisher.Connect("tcp://127.0.0.1:5556")
	time.Sleep(time.Second)
	data, _ := json.Marshal(&StreamReq{
		AccountID: account_id,
	})
	msg := fmt.Sprintf("%s %s %s", "STREAM_REQ", "OPEN", data)
	publisher.SendMessage(msg)
	fmt.Println(msg)

	// Give it some time
	time.Sleep(time.Second)

	// Listen
	stream, err := r.GetStream(account_id)

	if err != nil {
		return nil, err
	}

	return stream, nil
}

func (r *Replay) Close(account_id int, auth_code int) error {
	/*
		auth, err := r.Auth(account_id, auth_code)

		if err != nil {
			return nil, err
		}

		if auth == false {
			return nil, ErrAuthUnauthorized
		}
	*/

	// Broadcast request to start streaming
	publisher, err := zmq.NewSocket(zmq.PUB)
	if err != nil {
		panic(err)
	}
	publisher.SetIdentity("ro-replay")
	publisher.Connect("tcp://127.0.0.1:5556")
	time.Sleep(time.Second)
	data, _ := json.Marshal(&StreamReq{
		AccountID: account_id,
	})
	msg := fmt.Sprintf("%s %s %s", "STREAM_REQ", "CLOSE", data)
	publisher.SendMessage(msg)
	fmt.Println(msg)

	// Give it some time
	time.Sleep(time.Second)

	return nil
}

// Check with the authentication server if this is a valid auth_code given the account_id
func (r *Replay) Auth(account_id int, auth_code int) (bool, error) {
	/*
			auth := AuthCheck{account_id, auth_code}

			buf, err := json.Marshal(auth)

			if err != nil {
				return false, err
			}

			reply, err := r.AuthServer.Request(buf)

			if err != nil {
				return false, ErrAuthServerDown
			}

		resp := &AuthResp{}

		err := json.Unmarshal(buf, &resp)

		if err != nil {
			return false, err
		}

		if resp.Auth == true {
			return true, nil
		}
	*/
	return false, nil
}

// Get the stream of packets from the game proxy
func (r *Replay) GetStream(account_id int) (chan string, error) {
	// First publish to the proxy network to open a stream/subscription to this player
	subscriber, err := zmq.NewSocket(zmq.SUB)
	if err != nil {
		panic(err)
	}
	subscriber.Connect("tcp://127.0.0.1:5555")
	channel := fmt.Sprintf("STREAM_%d", account_id)
	subscriber.SetSubscribe(channel)

	stream := make(chan string, 100)

	go func() {
		defer subscriber.Close()
		for {
			msg, err := subscriber.Recv(0)
			if err != nil {
				close(stream)
				panic(err)
			}
			stream <- msg
		}
	}()

	return stream, nil
}

// Start publishing frm a channel
func (r *Replay) Publish(stream_id int, stream chan string) error {
	return nil
}

// Locate and read stream
func (r *Replay) Subscribe(stream_id int) (chan string, error) {
	return r.GetStream(stream_id)
}
