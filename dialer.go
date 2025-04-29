package kubedialer

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

type KubeDialer struct {
	client     kubernetes.Interface
	restConfig *rest.Config

	Logger Logger
}

func New() (*KubeDialer, error) {
	kubeconfigDir := filepath.Join(homeDir(), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigDir)
	if err != nil {
		return nil, err
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &KubeDialer{
		client:     client,
		restConfig: config,
		Logger:     &defaultLogger{},
	}, nil
}

func (d *KubeDialer) DialService(ctx context.Context, addr string) (net.Conn, error) {
	serviceName, port, err := net.SplitHostPort(addr)
	if err != nil {
		d.Logger.Errorf("invalid addr %s: %v", addr, err)
		return nil, err
	}
	paths := strings.Split(serviceName, ".")
	var name = paths[0]
	var namespace = "default"
	if len(paths) >= 2 {
		namespace = paths[1]
	}

	service, err := d.client.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		d.Logger.Errorf("failed to get service %s: %v", serviceName, err)
		return nil, err
	}
	selector := service.Spec.Selector
	if len(selector) == 0 {
		d.Logger.Errorf("service %s has no selectors", serviceName)
		return nil, fmt.Errorf("service \"%s\" has no selectors", serviceName)
	}
	labelSelector := metav1.FormatLabelSelector(&metav1.LabelSelector{MatchLabels: selector})
	pods, err := d.client.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, err
	}
	if len(pods.Items) == 0 {
		d.Logger.Errorf("service %s has no pods", serviceName)
		return nil, fmt.Errorf("no pods")
	}
	podIndex, err := rand.Int(rand.Reader, big.NewInt(1024*1024))
	if err != nil {
		d.Logger.Errorf("got unexpected error: %v", err)
		return nil, err
	}
	pod := pods.Items[int(podIndex.Int64())%len(pods.Items)]
	return d.DialPod(ctx, namespace, pod.Name, port)
}

func (d *KubeDialer) DialPod(ctx context.Context, namespace, podName, port string) (net.Conn, error) {
	pf := d.client.CoreV1().RESTClient().Post().Namespace(namespace).Resource("pods").
		Name(podName).SubResource("portforward")

	transport, upgrader, err := spdy.RoundTripperFor(d.restConfig)
	if err != nil {
		return nil, fmt.Errorf("RoundTripperFor: %w", err)
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", pf.URL())
	conn, _, err := dialer.Dial(portforward.PortForwardProtocolV1Name)
	if err != nil {
		return nil, fmt.Errorf("Dial: %w", err)
	}

	headers := http.Header{}
	headers.Set(corev1.StreamType, corev1.StreamTypeError)
	headers.Set(corev1.PortHeader, port)
	headers.Set(corev1.PortForwardRequestIDHeader, "kubeconn")
	errStream, err := conn.CreateStream(headers)
	if err != nil {
		return nil, fmt.Errorf("CreateStream: %w", err)
	}
	go func() {
		message, err := io.ReadAll(errStream)
		if err != nil {
			d.Logger.Errorf("got error when read error stream: %v", err)
			return
		}
		if len(message) > 0 {
			d.Logger.Errorf("got error from error stream: %s", string(message))
		}
	}()

	headers.Set(corev1.StreamType, corev1.StreamTypeData)
	stream, err := conn.CreateStream(headers)
	if err != nil {
		return nil, fmt.Errorf("CreateStream: %w", err)
	}
	return &connWrapper{stream, conn}, nil
}

type connWrapper struct {
	httpstream.Stream

	conn httpstream.Connection
}

func (c *connWrapper) LocalAddr() net.Addr                { return dummyAddr{} }
func (c *connWrapper) RemoteAddr() net.Addr               { return dummyAddr{} }
func (c *connWrapper) SetDeadline(t time.Time) error      { return nil }
func (c *connWrapper) SetReadDeadline(t time.Time) error  { return nil }
func (c *connWrapper) SetWriteDeadline(t time.Time) error { return nil }

func (c *connWrapper) Close() error {
	c.Stream.Reset()
	c.Stream.Close()
	c.conn.RemoveStreams(c.Stream)
	return c.conn.Close()
}

type dummyAddr struct{}

func (d dummyAddr) Network() string { return "spdy" }
func (d dummyAddr) String() string  { return "spdy-conn" }

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

type Logger interface {
	Debugf(format string, fields ...any)
	Infof(format string, fields ...any)
	Warnf(format string, fields ...any)
	Errorf(format string, fields ...any)
}

type defaultLogger struct{}

func (d *defaultLogger) Debugf(format string, fields ...any) {
	fmt.Printf("[DEBUG] "+format+"\n", fields...)
}

func (d *defaultLogger) Errorf(format string, fields ...any) {
	fmt.Printf("[ERROR] "+format+"\n", fields...)
}

func (d *defaultLogger) Infof(format string, fields ...any) {
	fmt.Printf("[INFO] "+format+"\n", fields...)
}

func (d *defaultLogger) Warnf(format string, fields ...any) {
	fmt.Printf("[WARN] "+format+"\n", fields...)
}

var _ Logger = (*defaultLogger)(nil)
