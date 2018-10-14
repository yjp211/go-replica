package quic
import(
	gquic "github.com/lucas-clemente/quic-go"
	"time"
	"net"
	"crypto/tls"
)

//封装quic-go的Session，对外暴露net.Conn的接口
//quic-go Session支持多路复用，我们主要用quic的拥塞控制，抛弃该特性进行封装
type QConn struct {
	sess gquic.Session
	mainStream gquic.Stream //只取其中的一个stream用于通信
}


func Dial(addr string, timeout time.Duration)(*QConn, error){
	session, err := gquic.DialAddr(addr,
		&tls.Config{InsecureSkipVerify: true},
		&gquic.Config{ HandshakeTimeout:timeout, IdleTimeout: timeout, KeepAlive: true})
	if err != nil {
		return nil, err
	}
	stream, err := session.OpenStreamSync()
	if err != nil {
		return nil, err
	}
	return &QConn{
		sess: session,
		mainStream: stream,
	}, nil
}

func (p *QConn) Read(b []byte) (n int, err error) {
	return p.mainStream.Read(b)
}


func (p *QConn) Write(b []byte) (n int, err error) {
	return p.mainStream.Write(b)
}


func (p *QConn) Close() error {
	p.mainStream.Close()
	return p.sess.Close()
}


func (p *QConn) LocalAddr() net.Addr {
	return p.sess.LocalAddr()
}


func (p *QConn) RemoteAddr() net.Addr {
	return p.sess.RemoteAddr()
}

func (p *QConn) SetDeadline(t time.Time) error {
	return p.mainStream.SetDeadline(t)
}

func (p *QConn) SetReadDeadline(t time.Time) error {
	return p.mainStream.SetReadDeadline(t)
}

func (p *QConn) SetWriteDeadline(t time.Time) error {
	return p.mainStream.SetWriteDeadline(t)
}


