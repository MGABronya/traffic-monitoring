package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/shirou/gopsutil/net"
	"github.com/shirou/gopsutil/process"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

// 存储每个端口流量情况的map
var Received map[uint32]([]uint64) = map[uint32]([]uint64){}
var Sent map[uint32]([]uint64) = map[uint32]([]uint64){}

// 防止读写冲突的锁
var lock sync.Mutex

// 将端口流量信息写入map
func checkPortTraffic(port uint32) {
	lock.Lock()
	Received[port] = make([]uint64, 0)
	Sent[port] = make([]uint64, 0)
	lock.Unlock()
	for {
		// 获取特定端口的进程ID
		pid, err := getPIDByPort(port)
		if err != nil {
			fmt.Println(port, ": Error getting PID:", err)
			return
		}

		// 获取进程的网络流量统计信息
		proc, err := process.NewProcess(int32(pid))
		if err != nil {
			fmt.Println(port, ": Error getting process:", err)
			return
		}
		ioCounters, err := proc.IOCounters()
		if err != nil {
			fmt.Println(port, ": Error getting IO counters:", err)
			return
		}

		// 打印接收和发送的字节数
		fmt.Printf("%d: Received bytes: %d, Sent bytes: %d\n", port, ioCounters.ReadBytes, ioCounters.WriteBytes)

		// 上锁写入
		lock.Lock()
		Received[port] = append(Received[port], ioCounters.ReadBytes)
		Sent[port] = append(Sent[port], ioCounters.WriteBytes)
		lock.Unlock()

		time.Sleep(1 * time.Second)
	}
}

// 获取端口的进程id
func getPIDByPort(port uint32) (int32, error) {
	// 获取所有网络连接信息
	conns, err := net.Connections("all")
	if err != nil {
		return 0, err
	}

	// 遍历连接信息，查找特定端口的进程ID
	for _, conn := range conns {
		if conn.Laddr.Port == port {
			return conn.Pid, nil
		}
	}

	return 0, fmt.Errorf("%d: No process found for port %d", port, port)
}

// 接收一个端口，将该端口的流量信息写入map
func test(a uint32) {
	port := uint32(a)
	checkPortTraffic(port)
}

func main() {
	// 获取所有网络连接信息
	conns, err := net.Connections("all")
	if err != nil {
		return
	}

	// 遍历连接信息，查找特定端口的进程ID
	for _, conn := range conns {
		go test(conn.Laddr.Port)
	}
	time.Sleep(60 * time.Second)

	// 绘图
	for k, a := range Received {
		// 流量为0不进入绘图
		if len(a) == 0 || a[0] == a[len(a)-1] {
			continue
		}
		p := plot.New()

		// 标题和xy轴
		p.Title.Text = "流量统计"
		p.X.Label.Text = "时间/1s"
		p.Y.Label.Text = "流量/1kb"
		points := make(plotter.XYs, len(a)-1)
		for i := 1; i < len(a); i++ {
			points[i-1].X = float64(i)
			points[i-1].Y = float64(a[i]-a[i-1]) / 1024
		}
		err = plotutil.AddLinePoints(p, fmt.Sprintf("%d-Received", k), points)
		if err != nil {
			log.Fatal(err)
		}
		if err = p.Save(32*vg.Inch, 16*vg.Inch, fmt.Sprintf("%d-Received.png", k)); err != nil {
			log.Fatal(err)
		}
	}
	for k, a := range Sent {
		// 流量为0不进入绘图
		if len(a) == 0 || a[0] == a[len(a)-1] {
			continue
		}
		p := plot.New()

		// 标题和xy轴
		p.Title.Text = "流量统计"
		p.X.Label.Text = "时间/1s"
		p.Y.Label.Text = "流量/1kb"
		points := make(plotter.XYs, len(a)-1)
		for i := 1; i < len(a); i++ {
			points[i-1].X = float64(i)
			points[i-1].Y = float64(a[i]-a[i-1]) / 1024
		}
		err = plotutil.AddLinePoints(p, fmt.Sprintf("%d-Sent", k), points)
		if err != nil {
			log.Fatal(err)
		}
		if err = p.Save(32*vg.Inch, 16*vg.Inch, fmt.Sprintf("%d-Sent.png", k)); err != nil {
			log.Fatal(err)
		}
	}
}
