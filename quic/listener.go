package quic

import(
	gquic "github.com/lucas-clemente/quic-go"
	"net"
)

//封装quic-go的listener 返回net.Listener格式
type QListener struct {
	listener gquic.Listener
}


func ListenAddr(addr string) (*QListener, error) {
	lis, err := gquic.ListenAddr(addr, TlsConfig, nil)
	if err != nil {
		return nil, err
	}

	qlis := &QListener{
		listener: lis,
	}

	return qlis, nil
}

func (p *QListener) Accept() (net.Conn, error){
	sess, err := p.listener.Accept()
	if err != nil{
		return nil, err
	}
	stream, err := sess.AcceptStream()
	if err != nil{
		return nil, err
	}
	conn := &QConn{
		sess: sess,
		mainStream: stream,
	}

	return conn, nil
}


func (p *QListener) Close() error{
	return p.listener.Close()
}


func (p *QListener) Addr() net.Addr{
	return p.listener.Addr()
}


