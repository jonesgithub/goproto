package generator

/*
There are three package type which is SimplePacket, VLFPacket and Packet

SimplePacket just is a packet header, not contains other field.
define SimplePacket like this:

// @SimplePacket: PKTTYPE_VIRDISKFINDBS, 0x00000001
type DiscoverBarServerRequest struct {}

VLFPacket is a slice packet.
define VLFPacket like this:

type ImageInfo struct {
	ImageName string
	ImageSize uint32
	NumSnapshot uint32
}

// @SimplePacket: PKTTYPE_QUERY_ALL_IMAGE_INFOS, 0x00000002
type QueryAllImageInfos struct {}

// @VLFPacket: PKTTYPE_QUERY_ALL_IMAGE_INFOS_ACK, 0x80000002
type QueryAllImageInfosAck struct {
	Infos []ImageInfo
}
If you define other fields in VLFPacket struct, like QueryAllImageInfosAck, they will be ignored.

Packet is general packet
define Packet like this:

// @Packet: PKTTYPE_VIRDISKLOGIN, 0x00000003
type LoginRequest struct {
	DiskType	uint32
	DiskID		uint32
	LoginReason uint32
}

as you can see, the comment format like this: <PacketType>:<PacketIDName>,<PacketID>
PacketType must is @SimplePacket, @VLFPacket, @Packet
PacketIDName is a ID's identifier
PacketID is a hex-based value.

Now just support follows data type:
	byte, int8, int16, int32, int64, uint8,uint16,uint32,uint64, string,
	struct, one-dimensional slice, one-dimensional array.
	Note: Now not support []struct and []string

*/
