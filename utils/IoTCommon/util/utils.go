package util

import (
	"errors"
	"fmt"
	"math"
	"net"
	"os"
	"reflect"
	"time"
)

const NoTTargetIP = "1.1.1.1"

func GetHostIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				if ipnet.IP.To4().String() == NoTTargetIP {
					continue
				}
				return ipnet.IP.To4().String()
			}
		}
	}

	return ""

	//netInterfaces, err := net.Interfaces()
	//if err != nil {
	//	fmt.Println("net.Interfaces failed, err:", err.Error())
	//	return ""
	//}
	//
	//var result string
	//for i := 0; i < len(netInterfaces); i++ {
	//
	//	inter := netInterfaces[i]
	//	if (inter.Flags&net.FlagUp) != 0 && (inter.Flags&net.FlagBroadcast != 0) {
	//		addrs, _ := netInterfaces[i].Addrs()
	//
	//		for _, address := range addrs {
	//			if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
	//
	//				if ipnet.IP.To4() != nil {
	//					result = ipnet.IP.String()
	//				}
	//			}
	//		}
	//	}
	//}
	//
	//return result
}

func GetAddrs() (map[string]string, error) {
	netWorkCardAddrsMap := make(map[string]string)
	netCard, err := net.Interfaces()
	if err != nil {
		return netWorkCardAddrsMap, err
	}
	for _, v := range netCard {
		addrs, err := v.Addrs()
		if err != nil {
			return netWorkCardAddrsMap, err
		}
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok {
				if ipnet.IP.To4() != nil {
					netWorkCardAddrsMap[v.Name] = ipnet.IP.String()
				}
			}
		}
	}
	return netWorkCardAddrsMap, err
}

func GetIPFromNetCard(netcard string) (string, error) {
	ipMap, ipMapErr := GetAddrs()
	if ipMapErr != nil {
		return "", ipMapErr
	}
	return ipMap[netcard], nil
}

func GetHost2Ip(host, netcard string) (string, error) {

	// 1、如果host存在，则取host对应的ip
	if host != "" {
		addrs, err := net.LookupHost(host)
		if len(addrs) == 0 {
			return "", fmt.Errorf("can't convert host -> %v to ip", host)
		}
		return addrs[0], err
	}

	// 2、如果host不存在，netcard存在，则取netcard对应的ip
	if host == "" && netcard != "" {
		netCardIp, netErr := func(netcard string) (string, error) {
			ipMap, ipMapErr := GetAddrs()
			return ipMap[netcard], ipMapErr
		}(netcard)
		if netErr != nil {
			return "", netErr
		}
		return netCardIp, nil
	}

	// 3、如果host、和netcard都没传，则
	//			a、去本机hostname
	//			b、调用LookupHost查找hostname对应的ip
	if host == "" && netcard == "" {
		hostname, hostnameErr := os.Hostname()
		if hostnameErr != nil {
			return "", hostnameErr
		}
		fmt.Printf("hostname:%v\n", hostname)
		addrs, err := net.LookupHost(hostname)
		if len(addrs) == 0 {
			return "", fmt.Errorf("can't convert host -> %v to ip", host)
		}
		return addrs[0], err
	}

	addrs, err := net.LookupHost(host)
	if len(addrs) == 0 {
		return "", fmt.Errorf("can't convert host -> %v to ip", host)
	}

	return addrs[0], err
}

// 删除指定元素
func Remove(slice []interface{}, elem interface{}) []interface{} {
	if len(slice) == 0 {
		return slice
	}
	for i, v := range slice {
		if v == elem {
			slice = append(slice[:i], slice[i+1:]...)
			return Remove(slice, elem)
		}
	}
	return slice
}

func RemoveByIndex(slice interface{}, index int) (interface{}, error) {
	sliceValue := reflect.ValueOf(slice)
	length := sliceValue.Len()
	if slice == nil || length == 0 || (length-1) < index {
		return nil, errors.New("error")
	}
	if length-1 == index {
		return sliceValue.Slice(0, index).Interface(), nil
	} else if (length - 1) >= index {
		return reflect.AppendSlice(sliceValue.Slice(0, index), sliceValue.Slice(index+1, length)).Interface(), nil
	}
	return nil, errors.New("error")

}

// 删除为零值的指定元素
func RemoveZero(slice []interface{}) []interface{} {
	if len(slice) == 0 {
		return slice
	}
	for i, v := range slice {
		if IfZero(v) {
			slice = append(slice[:i], slice[i+1:]...)
			return RemoveZero(slice)
		}
	}
	return slice
}

//判断一个值是否为零值，只支持string,float,int,time 以及其各自的指针，"%"和"%%"也属于零值范畴，场景是like语句
func IfZero(arg interface{}) bool {
	if arg == nil {
		return true
	}
	switch v := arg.(type) {
	case int, int32, int16, int64:
		if v == 0 {
			return true
		}
	case float32:
		r := float64(v)
		return math.Abs(r-0) < 0.0000001
	case float64:
		return math.Abs(v-0) < 0.0000001
	case string:
		if v == "" || v == "%%" || v == "%" {
			return true
		}
	case *string, *int, *int64, *int32, *int16, *int8, *float32, *float64, *time.Time:
		if v == nil {
			return true
		}
	case time.Time:
		return v.IsZero()
	default:
		return false
	}
	return false
}
