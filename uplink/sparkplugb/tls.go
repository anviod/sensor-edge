package sparkplugb

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"sync/atomic"
	"time"
)

type TLSReloader struct {
	config                    atomic.Value // *tls.Config
	caFile, certFile, keyFile string
	keyPassword               string
}

func NewTLSReloader(ca, cert, key, keyPwd string) *TLSReloader {
	r := &TLSReloader{caFile: ca, certFile: cert, keyFile: key, keyPassword: keyPwd}
	r.reload()
	go r.watch()
	return r
}

func (r *TLSReloader) reload() {
	cert, _ := tls.LoadX509KeyPair(r.certFile, r.keyFile)
	caCert, _ := ioutil.ReadFile(r.caFile)
	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(caCert)
	cfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caPool,
	}
	r.config.Store(cfg)
}

func (r *TLSReloader) watch() {
	for {
		time.Sleep(10 * time.Second)
		// 可监听文件变动，简化为定时 reload
		r.reload()
	}
}

func (r *TLSReloader) GetConfig() *tls.Config {
	return r.config.Load().(*tls.Config)
}
