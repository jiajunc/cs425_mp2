
package mp2util

import (
    "log"
    "math/rand"
    "net"
    "os"
    "time"
    // "fmt"
    // "strings"
)

var LocalIP = GetLocalIP()

type memberID struct {
    LocalIP    string
    JoinedTime time.Time
}

func GetLocalIP() net.IP {
    conn, err := net.Dial("udp", "8.8.8.8:80")
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    localAddr := conn.LocalAddr().(*net.UDPAddr)

    return localAddr.IP
}

func Shuffle(intSlice *[]int) {
    for i := len(*intSlice); i > 0; i-- {
        j := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(i)

        (*intSlice)[i-1], (*intSlice)[j] = (*intSlice)[j], (*intSlice)[i-1]
    }
}

func FileExists(name string) bool {
    if _, err := os.Stat(name); err != nil {
        if os.IsNotExist(err) {
            return false
        }
    }
    return true
}

func CreateFile(name string) error {
    fo, err := os.Create(name)
    if err != nil {
        return err
    }
    defer func() {
        fo.Close()
    }()
    return nil
}

func WriteLog(line string) error {
    if !FileExists("membership.log") {
        CreateFile("membership.log")
    }
    //create your file with desired read/write permissions
    f, err := os.OpenFile("membership.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
    if err != nil {
        log.Fatal(err)
    }
    //defer to close when you're done with it, not because you think it's idiomatic!
    defer f.Close()
    //set output of logs to f
    log.SetOutput(f)
    //test case
    log.Println(line)
    return nil
}
