package geecache

import pb "GeeCache/geecachepb"

// PeerPicker is the interface that must be implemented to locate
// the peer that owns a specific key.
type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// PeerGetter is the interface that must be implemented by a peer.
type PeerGetter interface {

	//Get(group string, key string) ([]byte, error)
	Get(in *pb.Request, out *pb.Response) error
}
