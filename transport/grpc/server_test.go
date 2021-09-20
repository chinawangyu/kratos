package grpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"google.golang.org/grpc"

	"github.com/stretchr/testify/assert"
)

type testKey struct{}

func randAddr() ServerOption {
	return Address(fmt.Sprintf("0.0.0.0:%d", 49152+rand.Intn(65535-49152)))
}

func TestServer(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, testKey{}, "test")
	srv := NewServer(
		randAddr(),
		Middleware([]middleware.Middleware{
			func(middleware.Handler) middleware.Handler { return nil },
		}...))

	if e, err := srv.Endpoint(); err != nil || e == nil || strings.HasSuffix(e.Host, ":0") {
		t.Fatal(e, err)
	}

	go func() {
		// start server
		if err := srv.Start(ctx); err != nil {
			panic(err)
		}
	}()
	time.Sleep(time.Second)
	testClient(t, srv)
	_ = srv.Stop(ctx)
}

func testClient(t *testing.T, srv *Server) {
	u, err := srv.Endpoint()
	if err != nil {
		t.Fatal(err)
	}
	// new a gRPC client
	conn, err := DialInsecure(context.Background(), WithEndpoint(u.Host))
	if err != nil {
		t.Fatal(err)
	}
	_ = conn.Close()
}

func TestNetwork(t *testing.T) {
	o := &Server{}
	v := "abc"
	Network(v)(o)
	assert.Equal(t, v, o.network)
}

func TestAddress(t *testing.T) {
	o := &Server{}
	v := "abc"
	Address(v)(o)
	assert.Equal(t, v, o.address)
}

func TestTimeout(t *testing.T) {
	o := &Server{}
	v := time.Duration(123)
	Timeout(v)(o)
	assert.Equal(t, v, o.timeout)
}

func TestMiddleware(t *testing.T) {
	o := &Server{}
	v := []middleware.Middleware{
		func(middleware.Handler) middleware.Handler { return nil },
	}
	Middleware(v...)(o)
	assert.Equal(t, v, o.middleware)
}

type mockLogger struct {
	level log.Level
	key   string
	val   string
}

func (l *mockLogger) Log(level log.Level, keyvals ...interface{}) error {
	l.level = level
	l.key = keyvals[0].(string)
	l.val = keyvals[1].(string)
	return nil
}

func TestLogger(t *testing.T) {
	o := &Server{}
	v := &mockLogger{}
	Logger(v)(o)
	o.log.Log(log.LevelWarn, "foo", "bar")
	assert.Equal(t, "foo", v.key)
	assert.Equal(t, "bar", v.val)
	assert.Equal(t, log.LevelWarn, v.level)
}

func TestTLSConfig(t *testing.T) {
	o := &Server{}
	v := &tls.Config{}
	TLSConfig(v)(o)
	assert.Equal(t, v, o.tlsConf)
}

func TestUnaryInterceptor(t *testing.T) {
	o := &Server{}
	v := []grpc.UnaryServerInterceptor{
		func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
			return nil, nil
		},
		func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
			return nil, nil
		},
	}
	UnaryInterceptor(v...)(o)
	assert.Equal(t, v, o.ints)
}

func TestOptions(t *testing.T) {
	o := &Server{}
	v := []grpc.ServerOption{
		grpc.EmptyServerOption{},
	}
	Options(v...)(o)
	assert.Equal(t, v, o.grpcOpts)
}

type testResp struct {
	Data string
}

func TestServer_unaryServerInterceptor(t *testing.T) {
	u, err := url.Parse("grpc://hello/world")
	assert.NoError(t, err)
	srv := &Server{
		baseCtx:    context.Background(),
		endpoint:   u,
		middleware: []middleware.Middleware{EmptyMiddleware()},
		timeout:    time.Duration(10),
	}
	req := &struct{}{}
	rv, err := srv.unaryServerInterceptor()(context.TODO(), req, &grpc.UnaryServerInfo{}, func(ctx context.Context, req interface{}) (i interface{}, e error) {
		return &testResp{Data: "hi"}, nil
	})
	assert.NoError(t, err)
	assert.Equal(t, "hi", rv.(*testResp).Data)
}

func TestServerAddress(t *testing.T) {
	s := NewServer(Address("0.0.0.0:8000"))
	e, err := s.Endpoint()
	assert.Nil(t, err)
	host, port, err := net.SplitHostPort(e.Host)
	assert.Nil(t, err)
	assert.Equal(t, "8000", port)
	ip := net.ParseIP(host)
	assert.NotNil(t, ip)
}

type mockAddr struct {
	addr string
}

func (a mockAddr) Network() string {
	return "tcp"
}

func (a mockAddr) String() string {
	return a.addr
}

type mockListener struct {
	addr string
}

func (l *mockListener) Accept() (c net.Conn, err error) {
	return
}

func (l *mockListener) Close() (err error) {
	return
}

func (l *mockListener) Addr() net.Addr {
	return mockAddr{addr: l.addr}
}

func TestServerListener(t *testing.T) {
	s := NewServer(Listener(&mockListener{":8090"}))
	e, err := s.Endpoint()
	assert.Nil(t, err)
	host, port, err := net.SplitHostPort(e.Host)
	assert.Nil(t, err)
	assert.Equal(t, "8090", port)
	ip := net.ParseIP(host)
	assert.NotNil(t, ip)
}
