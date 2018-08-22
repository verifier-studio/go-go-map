package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// 当前时间
var nowTime string = time.Now().Format("20060102150405")

// 阻塞队列
var waitgroup sync.WaitGroup

// 地球半径
var R float64 = 6378137

func main() {
	ColorPrintln("****************************", 2)
	ColorPrintln("Power by seaweedman studio.", 6)
	ColorPrintln("****************************", 2)
	fmt.Println()

	var mins, maxs, confirm, ok, fls string

	fmt.Printf("输入最小、大层级（半角逗号隔开）：")
	fmt.Scanln(&fls)

	fmt.Printf("输入最小经、纬度（半角逗号隔开）：")
	fmt.Scanln(&mins)

	fmt.Printf("输入最大经、纬度（半角逗号隔开）：")
	fmt.Scanln(&maxs)

	if fls == "" || mins == "" || maxs == "" {
		fmt.Println("输入值为空！")
		return
	}

	// 切割字符串
	mina := strings.Split(mins, ",")
	maxa := strings.Split(maxs, ",")
	fla := strings.Split(fls, ",")

	minLng, _ := strconv.ParseFloat(mina[0], 64)
	minLat, _ := strconv.ParseFloat(mina[1], 64)
	maxLng, _ := strconv.ParseFloat(maxa[0], 64)
	maxLat, _ := strconv.ParseFloat(maxa[1], 64)

	startZ, _ := strconv.Atoi(fla[0])
	endZ, _ := strconv.Atoi(fla[1])

	// 取得地图中心坐标经纬度
	lngCen := (minLng + maxLng) / 2.0
	latCen := (minLat + maxLat) / 2.0

	fmt.Printf("数据输入正确，是否开始下载？（Y/n）：")
	fmt.Scanln(&confirm)

	if confirm == "n" {
		fmt.Println("程序退出！")
		return
	}

	ColorPrintln("---------------------下载开始---------------------", 5)

	// 开始执行时间
	startTime := time.Now()

	GetAllFloor(minLng, maxLng, minLat, maxLat, startZ, endZ)

	// 计算执行耗时
	allTime := time.Since(startTime)

	fmt.Println(allTime)

	// 复制预览文件
	// _, _ = CopyFile("./demo.html", "./"+nowTime+"/index.html")

	// 生成地图索引文件
	MkIndex(lngCen, latCen, fla[0], fla[1])

	fmt.Println("下载完成！")
	fmt.Printf("是否预览地图？（Y/n）：")

	fmt.Scanln(&ok)

	if ok == "n" {
		fmt.Println("程序退出！")
		return
	} else {
		cmd := exec.Command("cmd", "/C", "start ./"+nowTime+"/index.html")
		cmd.Run()
	}
}

// 获得所有层级瓦片图
func GetAllFloor(minLng float64, maxLng float64, minLat float64, maxLat float64, startZ int, endZ int) {
	// 创建存储瓦片图总文件夹
	os.MkdirAll("./"+nowTime+"/tiles", os.ModePerm)

	for z := startZ; z <= endZ; z++ {
		waitgroup.Add(1)
		go GetOneFloor(minLng, maxLng, minLat, maxLat, z, nowTime)
		fmt.Println(strconv.Itoa(z))
	}

	waitgroup.Wait()
}

// 获得一个层级瓦片图
func GetOneFloor(minLng float64, maxLng float64, minLat float64, maxLat float64, z int, nowTime string) {
	url := "http://online1.map.bdimg.com/tile/?qt=tile&styles=pl&scaler=1&udt=20180810&z=" + strconv.Itoa(z)

	minX, maxX, minY, maxY := GetBound(minLng, maxLng, minLat, maxLat, z)

	var path string
	var dir string

	fdir := "./" + nowTime + "/tiles/" + strconv.Itoa(z) + "/"

	os.Mkdir(fdir, os.ModePerm)

	for i := minX; i <= maxX; i++ {
		func(i int, minY int, maxY int) {
			dir = fdir + strconv.Itoa(i)
			os.Mkdir(dir, os.ModePerm)
			for j := minY; j <= maxY; j++ {
				path = url + "&x=" + strconv.Itoa(i) + "&y=" + strconv.Itoa(j)

				resp, _ := http.Get(path)
				body, _ := ioutil.ReadAll(resp.Body)
				out, _ := os.Create(dir + "/" + strconv.Itoa(j) + ".png")
				io.Copy(out, bytes.NewReader(body))
			}
		}(i, minY, maxY)
	}

	waitgroup.Done()
}

// 根据经纬度和层级转换瓦片图范围
func GetBound(minLng float64, maxLng float64, minLat float64, maxLat float64, z int) (minX int, maxX int, minY int, maxY int) {
	minX = int(math.Floor(math.Pow(2.0, float64(z-26)) * (math.Pi * minLng * R / 180.0)))
	maxX = int(math.Floor(math.Pow(2.0, float64(z-26)) * (math.Pi * maxLng * R / 180.0)))

	minY = int(math.Floor(math.Pow(2.0, float64(z-26)) * R * math.Log(math.Tan(math.Pi*minLat/180.0)+1.0/math.Cos(math.Pi*minLat/180.0))))
	maxY = int(math.Floor(math.Pow(2.0, float64(z-26)) * R * math.Log(math.Tan(math.Pi*maxLat/180.0)+1.0/math.Cos(math.Pi*maxLat/180.0))))

	return
}

// 复制文件
func CopyFile(src, dst string) (w int64, err error) {
	srcFile, err := os.Open(src)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	defer dstFile.Close()

	return io.Copy(dstFile, srcFile)
}

// 生成地图索引文件
func MkIndex(lngCen, latCen float64, startZ string, endZ string) {
	fname := "./" + nowTime + "/index.html"
	f, err := os.OpenFile(fname, os.O_CREATE|os.O_RDWR|os.O_APPEND, os.ModeAppend|os.ModePerm)
	if err != nil {
		fmt.Println(err)
	}

	content := `<!DOCTYPE html>
				<html>
					<head>
						<meta charset="utf-8">
						<title>地图预览</title>

						<link rel="stylesheet" href="../js/leaflet/leaflet.css" />
						<style>
							#map {height: 800px;}
						</style>
					</head>
					<body>
						<div id="map"></div>

						<script src="../js/leaflet/leaflet.js"></script>
						<script src="../js/proj4-compressed.js"></script>
						<script src="../js/proj4leaflet.js"></script>
						<script>
						var center = {
								lng: "` + strconv.FormatFloat(lngCen, 'f', -1, 64) + `",
								lat: "` + strconv.FormatFloat(latCen, 'f', -1, 64) + `"
							}

						// 百度坐标转换
						var crs = new L.Proj.CRS('EPSG:3395',
								   '+proj=merc +lon_0=0 +k=1 +x_0=0 +y_0=0 +datum=WGS84 +units=m +no_defs',
								   {
									   resolutions: function () {
										   level = 19
										   var res = [];
										   res[0] = Math.pow(2, 18);
										   for (var i = 1; i < level; i++) {
											   res[i] = Math.pow(2, (18 - i))
										   }
										   return res;
									   }(),
									   origin: [0, 0],
									   bounds: L.bounds([20037508.342789244, 0], [0, 20037508.342789244])
								   }),
									map = L.map('map', {
									   crs: crs
								   });

						   L.tileLayer('./tiles/{z}/{x}/{y}.png', {
							   maxZoom: ` + endZ + `,
							   minZoom: ` + startZ + `,
							   subdomains: [0,1,2],
							   tms: true
						   }).addTo(map);

						   new L.marker([center.lat, center.lng]).addTo(map);

						   map.setView([center.lat, center.lng], ` + startZ + `);
						</script>
					</body>
				</html>`

	f.WriteString(content)
	f.Close()
}

// 彩色输出
func ColorPrintln(s string, i int) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("SetConsoleTextAttribute")
	handle, _, _ := proc.Call(uintptr(syscall.Stdout), uintptr(i)) //12 Red light

	fmt.Println(s)

	handle, _, _ = proc.Call(uintptr(syscall.Stdout), uintptr(7)) //White dark
	CloseHandle := kernel32.NewProc("CloseHandle")
	CloseHandle.Call(handle)
}
