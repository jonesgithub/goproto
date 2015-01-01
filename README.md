goproto
=======

用Go语言实现的一个协议生成器
可以用GO语言的语法来定义协议，然后用该程序就可以输出为包含序列化和反序列化的代码。

生成出的序列化代码非常简单，没有做什么向前还是向后的兼容，只是根据当时自己项目的需要所写的一个工具。
这份代码只包含了解析和生成部分的代码，以及给出了ReadStream/WriteStream接口的实现。
解析和生成部分的代码可以作为一份参考，即如何使用go自带的parser来扩展自己的代码。
协议定义文件主要是通过注释标识出哪些结构体是信令，以及是何种格式的信令及其信令ID的名称及ID值。

信令的种类主要有三种：
* 简单信令，即只有信令头的结构体，自身不含有任何字段。
* 普通信令，即内部有字段的结构体
* 信令数组，即该信令是一个切片类型

信令种类标识方式：
在结构体上方添加注释，格式如下：
<PacketType>:<PacketIDName>,<PacketIDValue>
PacketType按照上面的种类对应如下：
* @SimplePacket，表示简单信令
* @Packet, 表示普通信令
* @VLFPacket, 表示信令数组

PacketIDName用于为PacketIDValue分配的名称
PacketIDValue表示该信令的ID

下面是一个简单的例子
```golang
package protocol

// @SimplePacket: KEEPALIVE_REQUEST, 0x00000001
type KeepaliveRequest struct {}

// @SimplePacket:KEEPALIVE_RESPONSE, 0x80000001
type KeepaliveResponse struct {}

// @Packet: LOGIN_REQUEST, 0x00000002
type LoginRequest struct {
    UserName string
    Password string    
}

// @Packet: LOGIN_RESPONSE, 0x80000002
type LoginResponse struct {
    Ack uint32
    SessionID uint64
}

// @Packet: QUERY_BUDDYLIST_REQUEST, 0x00000003
type QueryBuddyListRequest struct {
    SessionID uint64
}

type BuddyInfo struct {
    NickName string
    Age uint32
    Male uint32
    Address string
}

// @VLFPacket: QUERY_BUDDYLIST_RESPONSE, 0x80000003
type QueryBuddyListResponse struct {
    Infos []BuddyInfo
}

```
