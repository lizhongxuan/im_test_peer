package main

import "net"
import "time"
import "sync"
import (
	"fmt"
	log "github.com/golang/glog"
	"sync/atomic"
)

type Subscriber struct {
	uids     map[int64]int
	room_ids map[int64]int
}

func NewSubscriber() *Subscriber {
	s := new(Subscriber)
	s.uids = make(map[int64]int)
	s.room_ids = make(map[int64]int)
	return s
}

type Channel struct {
	addr        string
	wt          chan *Message
	mutex       sync.Mutex
	subscribers map[int64]*Subscriber
	dispatch    func(*AppMessage)
	uid         int64
	seqN        int
	seq int64
	sendM   int64
}

func NewChannel(addr string, uid int64) *Channel {
	if uid < 1000000 {
		log.Error("uid <1000000")
		return nil
	}

	channel := new(Channel)
	channel.subscribers = make(map[int64]*Subscriber)
	channel.addr = addr
	channel.uid = uid
	channel.wt = make(chan *Message, 10)
	return channel
}

func (client *Channel) RunOnce(conn *net.TCPConn) {
	defer conn.Close()
	go func() {
		//接收
		for {
			msg := ReceiveMessage(conn)
			client.HandleMessage(msg)
			if msg == nil {
				log.Info("client read end")
				return
			}

		}
	}()

	//用access_token 连接IM
	err := SendMessage(conn, &Message{
		cmd:     MSG_AUTH_TOKEN,
		seq:     0,
		version: 0,
		flag:    0,
		body: &AuthenticationToken{
			token:       fmt.Sprintf("%d",client.uid),
			device_id:   fmt.Sprintf("device_id_%d", client.uid),
			platform_id: 1,
		},
	})
	if err != nil {
		log.Info("用access_token 连接IM 失败 :", err)
		return
	}

	client.seqN++
	//向IM请求数据
	s := &Message{MSG_SYNC, client.seqN, DEFAULT_VERSION, 0, &SyncKey{client.seq}}
	client.wt <- s


	for msg:=range client.wt {

			if msg == nil {
				log.Info("client sent end")
				return
			}
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			err := SendMessage(conn, msg)
			if err != nil {
				log.Info("channel send message:", err)
			}
	}
	
	//发送
	for msg := range client.wt {
		if msg == nil {
			log.Info("client sent end")
			return
		}
		conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		err := SendMessage(conn, msg)
		if err != nil {
			log.Info("channel send message:", err)
		}
	}
}

func (client *Channel) HandleMessage(msg *Message) {
	//log.Info("------------->HandleMessage<--------------------")
	//log.Info("msg:", msg)
	if msg == nil {
		client.wt <- msg
		log.Error("----------  msg == nil  --->HandleMessage end .")
		return
	}

	log.Info("read msg cmd:", Command(msg.cmd))
	switch msg.cmd {
	case MSG_AUTH_STATUS:
		fmt.Println("status=", msg.body.(*AuthenticationStatus).status, "  ip=", msg.body.(*AuthenticationStatus).ip)
	case MSG_ACK:
		//log.Info("seq=", msg.body.(*MessageACK).seq)
	case MSG_HEARTBEAT:
		// nothing to do
	case MSG_PING:
		//client.HandlePing()
	case MSG_SYNC_NOTIFY:
		//log.Info("sync_key=", msg.body.(*SyncKey).sync_key)
		sync_key := msg.body.(*SyncKey).sync_key

		if sync_key > client.seq {
			client.seqN++
			//向IM请求数据
			s := &Message{MSG_SYNC, client.seqN, DEFAULT_VERSION, 0, &SyncKey{client.seq}}
			client.wt <- s
			client.seq = sync_key
			client.seqN++
			//更新sync_key
			sk := &Message{MSG_SYNC_KEY, client.seqN, DEFAULT_VERSION, 0, &SyncKey{client.seq}}
			client.wt <- sk
		}
		client.seqN++
		//返回个ACK保活
		ack := &Message{cmd: MSG_ACK, body: &MessageACK{int32(msg.seq)}}
		client.wt <- ack


	case MSG_IM:
		m := msg.body.(*IMMessage)
		//log.Infof("sender:%d receiver:%d content:%s", m.sender, m.receiver, m.content)

		if int64(m.msgid) > client.seq {
			client.seq = int64(m.msgid)
		}

		atomic.AddInt64(&receiveNum,1)
		//返回个ACK保活
		ack := &Message{cmd: MSG_ACK, body: &MessageACK{int32(msg.seq)}}
		client.wt <- ack
	case MSG_SYSTEM:
		//m := msg.body.(*SystemMessage)
		//log.Infof("notification:%s", m.notification)

		//返回个ACK保活
		//ack := &Message{cmd: MSG_ACK, body: &MessageACK{int32(msg.seq)}}
		//client.wt <- ack
	case MSG_GROUP_IM:
		m := msg.body.(*IMMessage)
		//log.Infof("sender:%d receiver:%d content:%s", m.sender, m.receiver, m.content)
		//client.seq = int64(m.msgid)

		if int64(m.msgid) > client.seq {
			client.seq = int64(m.msgid)
		}
		atomic.AddInt64(&receiveNum,1)
		//返回个ACK保活
		ack := &Message{cmd: MSG_ACK, body: &MessageACK{int32(msg.seq)}}
		client.wt <- ack
	}
}

func (channel *Channel) Run() {
	nsleep := 100
	for {
		conn, err := net.Dial("tcp", channel.addr)
		if err != nil {
			log.Info("connect route server error:", err)
			nsleep *= 2
			if nsleep > 60*1000 {
				nsleep = 60 * 1000
			}
			log.Info("channel sleep:", nsleep)
			time.Sleep(time.Duration(nsleep) * time.Millisecond)
			continue
		}
		tconn := conn.(*net.TCPConn)
		tconn.SetKeepAlive(true)
		tconn.SetKeepAlivePeriod(time.Duration(10 * 60 * time.Second))
		log.Info("channel connected")
		nsleep = 100
		channel.RunOnce(tconn)
	}
}

func (channel *Channel) Start() {
	channel.Run()
}