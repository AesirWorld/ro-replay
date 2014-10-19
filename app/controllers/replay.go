package controllers

import "code.google.com/p/go.net/websocket"
import "fmt"
import "ro-replay/app/replay"
import "github.com/robfig/revel"
import "database/sql"
import "net/http"
import "log"
import "io"
import "errors"
import "compress/zlib"

var r = &replay.Replay{}

type Replay struct {
	*revel.Controller
}

type ReplayRegisterResponse struct {
	StreamID int    `json:"stream_id"`
	Err      string `json:"err"`
}

type ReplayDataResponse struct {
	AID            int    `json:"account_id"`
	UID            string `json:"uid"`
	BufferID       string `json:"replay_buffer_id"`
	BufferURI      string `json:"replay_buffer_uri"`
	RecordingStart int    `json:"recording_start"`
	RecordingEnd   int    `json:"recording_end"`
	Err            string `json:"err"`
}

func (c *Replay) Data(uid string) revel.Result {
	var account_id int
	var buffer_id string
	var start int
	var end int

	row := Db.QueryRow("SELECT account_id, replay_buffer_id, recording_start, recording_end FROM ro_replays WHERE uid = ?", uid)

	err := row.Scan(&account_id, &buffer_id, &start, &end)

	switch {
	case err == sql.ErrNoRows:
		c.Response.Status = http.StatusNotFound
		return c.RenderJson(&ReplayDataResponse{
			Err: "Replay not found.",
		})
	case err != nil:
		log.Println(err)
		c.Response.Status = http.StatusInternalServerError
		return c.RenderJson(&ReplayDataResponse{
			Err: "Internal server error.",
		})
	default:
		return c.RenderJson(&ReplayDataResponse{
			AID:            account_id,
			UID:            uid,
			BufferID:       buffer_id,
			BufferURI:      "http://127.0.0.1/v1/replay/buffer/" + buffer_id,
			RecordingStart: start,
			RecordingEnd:   end,
		})
	}
}

type RenderReplay struct {
	reader io.ReadCloser
}

func (r RenderReplay) Apply(req *revel.Request, resp *revel.Response) {
	defer r.reader.Close()

	// Decompressed replay buffer
	reader, err := zlib.NewReader(r.reader)

	if err != nil {
		panic(err)
	}

	// Little hack: set to `text/plain` so CloudFlare caches and gzips this resource
	resp.WriteHeader(http.StatusOK, "text/plain")

	for {
		chunk := make([]byte, 1024)
		n, err := reader.Read(chunk)

		if n > 0 {
			resp.Out.Write(chunk)
		}

		if err == io.EOF {
			break
		} else if err != nil {
			log.Println(err)
			return
		}
	}
}

func (c *Replay) Buffer(uid string) revel.Result {
	blob_host, _ := revel.Config.String("db.endpoint") // Little hack, its temporary

	replay_uri := fmt.Sprintf("%s_blobs/ro_replays/%s", blob_host, uid)

	resp, err := http.Get(replay_uri)

	if err != nil {
		log.Println(err)
		c.Response.Status = http.StatusInternalServerError
		return c.RenderError(errors.New("Failed to fetch blob from storage"))
	}

	r := RenderReplay{
		reader: resp.Body,
	}

	return r
}
func (c *Replay) Register() revel.Result {
	var account_id int
	var auth_code int

	c.Params.Bind(&account_id, "account_id")
	c.Params.Bind(&auth_code, "auth_code")

	stream, err := r.Register(account_id, auth_code)

	if err != nil {
		switch err {
		case replay.ErrAuthServerDown:
			c.Response.Status = http.StatusServiceUnavailable
			return c.RenderJson(&ReplayRegisterResponse{
				Err: "Replay server not available at the moment.",
			})
		case replay.ErrAuthUnauthorized:
			c.Response.Status = http.StatusUnauthorized
			return c.RenderJson(&ReplayRegisterResponse{
				Err: "Not authroized.",
			})
		default:
			c.Response.Status = http.StatusInternalServerError
			return c.RenderJson(&ReplayRegisterResponse{
				Err: "Internal server error.",
			})
		}
		return nil
	}

	_ = r.Publish(account_id, stream)

	return c.RenderJson(&ReplayRegisterResponse{
		StreamID: account_id,
	})
}

func (c *Replay) Close() revel.Result {
	var account_id int
	var auth_code int

	c.Params.Bind(&account_id, "account_id")
	c.Params.Bind(&auth_code, "auth_code")

	err := r.Close(account_id, auth_code)

	if err != nil {
		switch err {
		case replay.ErrAuthServerDown:
			c.Response.Status = http.StatusServiceUnavailable
			return c.RenderJson(&ReplayRegisterResponse{
				Err: "Replay server not available at the moment.",
			})
		case replay.ErrAuthUnauthorized:
			c.Response.Status = http.StatusUnauthorized
			return c.RenderJson(&ReplayRegisterResponse{
				Err: "Not authroized.",
			})
		default:
			c.Response.Status = http.StatusInternalServerError
			return c.RenderJson(&ReplayRegisterResponse{
				Err: "Internal server error.",
			})
		}
		return nil
	}

	return c.RenderJson(&ReplayRegisterResponse{
		StreamID: account_id,
	})
}

func (c *Replay) Subscribe(stream_id int, ws *websocket.Conn) revel.Result {
	stream, err := r.Subscribe(stream_id)

	fmt.Println("WS stream_id", stream_id)

	if err != nil {
		switch err {
		default:
			c.Response.Status = http.StatusInternalServerError
			return c.RenderJson(&ReplayRegisterResponse{
				Err: "Internal server error.",
			})
		}
		return nil
	}

	for {
		select {
		case data, open := <-stream:
			if open == false {
				// Stream ended
				fmt.Println("Stream", stream_id, "closed")
				return nil
			}
			if websocket.Message.Send(ws, data) != nil {
				// Disconnected
				fmt.Println("WS disconnected")
				return nil
			}
		}
	}

	return nil
}
