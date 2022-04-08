package main

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

type AdapterWithIPNets struct {
	Name        string
	Description string
	Gateways    []string
	IPNets      []net.IPNet
}

// 获取有效的本地网络IP地址
func GetLocalNicIP() string {
	nics, err := Adapters()
	if err != nil {
		fmt.Println("Get Adapter Error:", err)
		return ""
	}

	for _, nic := range nics {
		if len(nic.Gateways) > 0 && nic.Gateways[0] != "0.0.0.0" {
			for _, ip := range nic.IPNets {
				if isPrivateIP(ip.IP) {
					return ip.IP.String()
				}
			}
		}

	}

	return ""
}

func Adapters() ([]AdapterWithIPNets, error) {
	var awins []AdapterWithIPNets
	ai, err := getAdapterList()
	if err != nil {
		return nil, err
	}
	for ; ai != nil; ai = ai.Next {
		description := bytePtrToString(&ai.Description[0])
		name := bytePtrToString(&ai.AdapterName[0])
		awin := AdapterWithIPNets{Name: name, Description: description}
		iai := &ai.IpAddressList
		//gateway := net.ParseIP(bytePtrToString(&ai.GatewayList.IpAddress.String[0]))
		for ; iai != nil; iai = iai.Next {
			ip := net.ParseIP(bytePtrToString(&iai.IpAddress.String[0]))
			mask := parseIPv4Mask(bytePtrToString(&iai.IpMask.String[0]))
			gateway := bytePtrToString(&ai.GatewayList.IpAddress.String[0])
			awin.Gateways = append(awin.Gateways, gateway)
			awin.IPNets = append(awin.IPNets, net.IPNet{IP: ip, Mask: mask})
		}
		awins = append(awins, awin)
	}
	return awins, nil
}

func parseIPv4Mask(ipStr string) net.IPMask {
	ip := net.ParseIP(ipStr).To4()
	return net.IPv4Mask(ip[0], ip[1], ip[2], ip[3])
}

// https://github.com/golang/go/blob/go1.4.1/src/net/interface_windows.go#L13-L20
func bytePtrToString(p *uint8) string {
	a := (*[10000]uint8)(unsafe.Pointer(p))
	i := 0
	for a[i] != 0 {
		i++
	}
	return string(a[:i])
}

// copied from https://github.com/golang/go/blob/go1.4.1/src/net/interface_windows.go#L22-L39
func getAdapterList() (*syscall.IpAdapterInfo, error) {
	b := make([]byte, 1000)
	l := uint32(len(b))
	a := (*syscall.IpAdapterInfo)(unsafe.Pointer(&b[0]))
	// TODO(mikio): GetAdaptersInfo returns IP_ADAPTER_INFO that
	// contains IPv4 address list only. We should use another API
	// for fetching IPv6 stuff from the kernel.
	err := syscall.GetAdaptersInfo(a, &l)
	if err == syscall.ERROR_BUFFER_OVERFLOW {
		b = make([]byte, l)
		a = (*syscall.IpAdapterInfo)(unsafe.Pointer(&b[0]))
		err = syscall.GetAdaptersInfo(a, &l)
	}
	if err != nil {
		return nil, os.NewSyscallError("GetAdaptersInfo", err)
	}
	return a, nil
}

func IsUp(nif *net.Interface) bool        { return nif.Flags&net.FlagUp != 0 }
func IsLoopback(nif *net.Interface) bool  { return nif.Flags&net.FlagLoopback != 0 }
func IsBroadcast(nif *net.Interface) bool { return nif.Flags&net.FlagBroadcast != 0 }

func IsProblematicInterface(nif *net.Interface) bool {
	name := nif.Name
	if strings.HasPrefix(name, "zt") || (runtime.GOOS == "windows" && (strings.Contains(name, "ZeroTier") || strings.Contains(name, "SSTAP"))) {
		return true
	}
	return false
}

// 根据syscall包获取网卡信息 (底层原理是通过windows dll api获取)
func GetAdapter() (*syscall.IpAdapterInfo, error) {
	bTmp := make([]byte, 15000)
	length := uint32(len(bTmp))
	adtr := (*syscall.IpAdapterInfo)(unsafe.Pointer(&bTmp[0]))

	if err := syscall.GetAdaptersInfo(adtr, &length); err == syscall.ERROR_BUFFER_OVERFLOW {
		bTmp = make([]byte, length)
		adtr = (*syscall.IpAdapterInfo)(unsafe.Pointer(&bTmp[0]))
		if err = syscall.GetAdaptersInfo(adtr, &length); err != nil {
			return nil, os.NewSyscallError("GetAdaptersInfo", err)
		}
	}
	return adtr, nil
}
