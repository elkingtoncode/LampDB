// Code generated by protoc-gen-gogo.
// source: cockroach/storage/engine/mvcc.proto
// DO NOT EDIT!

/*
	Package engine is a generated protocol buffer package.

	It is generated from these files:
		cockroach/storage/engine/mvcc.proto

	It has these top-level messages:
		MVCCValue
		MVCCMetadata
		MVCCStats
*/
package engine

import proto "github.com/gogo/protobuf/proto"
import math "math"
import cockroach_proto1 "github.com/cockroachdb/cockroach/proto"

// discarding unused import gogoproto "gogoproto"

import io "io"
import fmt "fmt"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = math.Inf

// MVCCValue differentiates between normal versioned values and
// deletion tombstones.
type MVCCValue struct {
	// True to indicate a deletion tombstone. If false, value should not
	// be nil.
	Deleted bool `protobuf:"varint,1,opt,name=deleted" json:"deleted"`
	// The value. Nil if deleted is true; not nil otherwise.
	Value            *cockroach_proto1.Value `protobuf:"bytes,2,opt,name=value" json:"value,omitempty"`
	XXX_unrecognized []byte                  `json:"-"`
}

func (m *MVCCValue) Reset()         { *m = MVCCValue{} }
func (m *MVCCValue) String() string { return proto.CompactTextString(m) }
func (*MVCCValue) ProtoMessage()    {}

func (m *MVCCValue) GetDeleted() bool {
	if m != nil {
		return m.Deleted
	}
	return false
}

func (m *MVCCValue) GetValue() *cockroach_proto1.Value {
	if m != nil {
		return m.Value
	}
	return nil
}

// MVCCMetadata holds MVCC metadata for a key. Used by storage/engine/mvcc.go.
type MVCCMetadata struct {
	Txn *cockroach_proto1.Transaction `protobuf:"bytes,1,opt,name=txn" json:"txn,omitempty"`
	// The timestamp of the most recent versioned value.
	Timestamp cockroach_proto1.Timestamp `protobuf:"bytes,2,opt,name=timestamp" json:"timestamp"`
	// Is the most recent value a deletion tombstone?
	Deleted bool `protobuf:"varint,3,opt,name=deleted" json:"deleted"`
	// The size in bytes of the most recent encoded key.
	KeyBytes int64 `protobuf:"varint,4,opt,name=key_bytes" json:"key_bytes"`
	// The size in bytes of the most recent versioned value.
	ValBytes int64 `protobuf:"varint,5,opt,name=val_bytes" json:"val_bytes"`
	// Inline value, used for values with zero timestamp. This provides
	// an efficient short circuit of the normal MVCC metadata sentinel
	// and subsequent version rows. If timestamp == (0, 0), then there
	// is only a single MVCC metadata row with value inlined, and with
	// empty timestamp, key_bytes, and val_bytes.
	Value            *cockroach_proto1.Value `protobuf:"bytes,6,opt,name=value" json:"value,omitempty"`
	XXX_unrecognized []byte                  `json:"-"`
}

func (m *MVCCMetadata) Reset()         { *m = MVCCMetadata{} }
func (m *MVCCMetadata) String() string { return proto.CompactTextString(m) }
func (*MVCCMetadata) ProtoMessage()    {}

func (m *MVCCMetadata) GetTxn() *cockroach_proto1.Transaction {
	if m != nil {
		return m.Txn
	}
	return nil
}

func (m *MVCCMetadata) GetTimestamp() cockroach_proto1.Timestamp {
	if m != nil {
		return m.Timestamp
	}
	return cockroach_proto1.Timestamp{}
}

func (m *MVCCMetadata) GetDeleted() bool {
	if m != nil {
		return m.Deleted
	}
	return false
}

func (m *MVCCMetadata) GetKeyBytes() int64 {
	if m != nil {
		return m.KeyBytes
	}
	return 0
}

func (m *MVCCMetadata) GetValBytes() int64 {
	if m != nil {
		return m.ValBytes
	}
	return 0
}

func (m *MVCCMetadata) GetValue() *cockroach_proto1.Value {
	if m != nil {
		return m.Value
	}
	return nil
}

// MVCCStats tracks byte and instance counts for:
//  - Live key/values (i.e. what a scan at current time will reveal;
//    note that this includes intent keys and values, but not keys and
//    values with most recent value deleted)
//  - Key bytes (includes all keys, even those with most recent value deleted)
//  - Value bytes (includes all versions)
//  - Key count (count of all keys, including keys with deleted tombstones)
//  - Value count (all versions, including deleted tombstones)
//  - Intents (provisional values written during txns)
//  - System-local key counts and byte totals
type MVCCStats struct {
	LiveBytes        int64  `protobuf:"varint,1,opt,name=live_bytes" json:"live_bytes"`
	KeyBytes         int64  `protobuf:"varint,2,opt,name=key_bytes" json:"key_bytes"`
	ValBytes         int64  `protobuf:"varint,3,opt,name=val_bytes" json:"val_bytes"`
	IntentBytes      int64  `protobuf:"varint,4,opt,name=intent_bytes" json:"intent_bytes"`
	LiveCount        int64  `protobuf:"varint,5,opt,name=live_count" json:"live_count"`
	KeyCount         int64  `protobuf:"varint,6,opt,name=key_count" json:"key_count"`
	ValCount         int64  `protobuf:"varint,7,opt,name=val_count" json:"val_count"`
	IntentCount      int64  `protobuf:"varint,8,opt,name=intent_count" json:"intent_count"`
	IntentAge        int64  `protobuf:"varint,9,opt,name=intent_age" json:"intent_age"`
	GCBytesAge       int64  `protobuf:"varint,10,opt,name=gc_bytes_age" json:"gc_bytes_age"`
	SysBytes         int64  `protobuf:"varint,12,opt,name=sys_bytes" json:"sys_bytes"`
	SysCount         int64  `protobuf:"varint,13,opt,name=sys_count" json:"sys_count"`
	LastUpdateNanos  int64  `protobuf:"varint,30,opt,name=last_update_nanos" json:"last_update_nanos"`
	XXX_unrecognized []byte `json:"-"`
}

func (m *MVCCStats) Reset()         { *m = MVCCStats{} }
func (m *MVCCStats) String() string { return proto.CompactTextString(m) }
func (*MVCCStats) ProtoMessage()    {}

func (m *MVCCStats) GetLiveBytes() int64 {
	if m != nil {
		return m.LiveBytes
	}
	return 0
}

func (m *MVCCStats) GetKeyBytes() int64 {
	if m != nil {
		return m.KeyBytes
	}
	return 0
}

func (m *MVCCStats) GetValBytes() int64 {
	if m != nil {
		return m.ValBytes
	}
	return 0
}

func (m *MVCCStats) GetIntentBytes() int64 {
	if m != nil {
		return m.IntentBytes
	}
	return 0
}

func (m *MVCCStats) GetLiveCount() int64 {
	if m != nil {
		return m.LiveCount
	}
	return 0
}

func (m *MVCCStats) GetKeyCount() int64 {
	if m != nil {
		return m.KeyCount
	}
	return 0
}

func (m *MVCCStats) GetValCount() int64 {
	if m != nil {
		return m.ValCount
	}
	return 0
}

func (m *MVCCStats) GetIntentCount() int64 {
	if m != nil {
		return m.IntentCount
	}
	return 0
}

func (m *MVCCStats) GetIntentAge() int64 {
	if m != nil {
		return m.IntentAge
	}
	return 0
}

func (m *MVCCStats) GetGCBytesAge() int64 {
	if m != nil {
		return m.GCBytesAge
	}
	return 0
}

func (m *MVCCStats) GetSysBytes() int64 {
	if m != nil {
		return m.SysBytes
	}
	return 0
}

func (m *MVCCStats) GetSysCount() int64 {
	if m != nil {
		return m.SysCount
	}
	return 0
}

func (m *MVCCStats) GetLastUpdateNanos() int64 {
	if m != nil {
		return m.LastUpdateNanos
	}
	return 0
}

func (m *MVCCValue) Unmarshal(data []byte) error {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Deleted", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				v |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.Deleted = bool(v != 0)
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Value", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			postIndex := iNdEx + msglen
			if msglen < 0 {
				return ErrInvalidLengthMvcc
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Value == nil {
				m.Value = &cockroach_proto1.Value{}
			}
			if err := m.Value.Unmarshal(data[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			var sizeOfWire int
			for {
				sizeOfWire++
				wire >>= 7
				if wire == 0 {
					break
				}
			}
			iNdEx -= sizeOfWire
			skippy, err := skipMvcc(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthMvcc
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, data[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	return nil
}
func (m *MVCCMetadata) Unmarshal(data []byte) error {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Txn", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			postIndex := iNdEx + msglen
			if msglen < 0 {
				return ErrInvalidLengthMvcc
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Txn == nil {
				m.Txn = &cockroach_proto1.Transaction{}
			}
			if err := m.Txn.Unmarshal(data[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Timestamp", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			postIndex := iNdEx + msglen
			if msglen < 0 {
				return ErrInvalidLengthMvcc
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Timestamp.Unmarshal(data[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Deleted", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				v |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.Deleted = bool(v != 0)
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field KeyBytes", wireType)
			}
			m.KeyBytes = 0
			for shift := uint(0); ; shift += 7 {
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				m.KeyBytes |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 5:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field ValBytes", wireType)
			}
			m.ValBytes = 0
			for shift := uint(0); ; shift += 7 {
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				m.ValBytes |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Value", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			postIndex := iNdEx + msglen
			if msglen < 0 {
				return ErrInvalidLengthMvcc
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Value == nil {
				m.Value = &cockroach_proto1.Value{}
			}
			if err := m.Value.Unmarshal(data[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			var sizeOfWire int
			for {
				sizeOfWire++
				wire >>= 7
				if wire == 0 {
					break
				}
			}
			iNdEx -= sizeOfWire
			skippy, err := skipMvcc(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthMvcc
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, data[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	return nil
}
func (m *MVCCStats) Unmarshal(data []byte) error {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field LiveBytes", wireType)
			}
			m.LiveBytes = 0
			for shift := uint(0); ; shift += 7 {
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				m.LiveBytes |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field KeyBytes", wireType)
			}
			m.KeyBytes = 0
			for shift := uint(0); ; shift += 7 {
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				m.KeyBytes |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field ValBytes", wireType)
			}
			m.ValBytes = 0
			for shift := uint(0); ; shift += 7 {
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				m.ValBytes |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field IntentBytes", wireType)
			}
			m.IntentBytes = 0
			for shift := uint(0); ; shift += 7 {
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				m.IntentBytes |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 5:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field LiveCount", wireType)
			}
			m.LiveCount = 0
			for shift := uint(0); ; shift += 7 {
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				m.LiveCount |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 6:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field KeyCount", wireType)
			}
			m.KeyCount = 0
			for shift := uint(0); ; shift += 7 {
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				m.KeyCount |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 7:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field ValCount", wireType)
			}
			m.ValCount = 0
			for shift := uint(0); ; shift += 7 {
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				m.ValCount |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 8:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field IntentCount", wireType)
			}
			m.IntentCount = 0
			for shift := uint(0); ; shift += 7 {
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				m.IntentCount |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 9:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field IntentAge", wireType)
			}
			m.IntentAge = 0
			for shift := uint(0); ; shift += 7 {
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				m.IntentAge |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 10:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field GCBytesAge", wireType)
			}
			m.GCBytesAge = 0
			for shift := uint(0); ; shift += 7 {
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				m.GCBytesAge |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 12:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field SysBytes", wireType)
			}
			m.SysBytes = 0
			for shift := uint(0); ; shift += 7 {
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				m.SysBytes |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 13:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field SysCount", wireType)
			}
			m.SysCount = 0
			for shift := uint(0); ; shift += 7 {
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				m.SysCount |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 30:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field LastUpdateNanos", wireType)
			}
			m.LastUpdateNanos = 0
			for shift := uint(0); ; shift += 7 {
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				m.LastUpdateNanos |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			var sizeOfWire int
			for {
				sizeOfWire++
				wire >>= 7
				if wire == 0 {
					break
				}
			}
			iNdEx -= sizeOfWire
			skippy, err := skipMvcc(data[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthMvcc
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, data[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	return nil
}
func skipMvcc(data []byte) (n int, err error) {
	l := len(data)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := data[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for {
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if data[iNdEx-1] < 0x80 {
					break
				}
			}
			return iNdEx, nil
		case 1:
			iNdEx += 8
			return iNdEx, nil
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := data[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			iNdEx += length
			if length < 0 {
				return 0, ErrInvalidLengthMvcc
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if iNdEx >= l {
						return 0, io.ErrUnexpectedEOF
					}
					b := data[iNdEx]
					iNdEx++
					innerWire |= (uint64(b) & 0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				innerWireType := int(innerWire & 0x7)
				if innerWireType == 4 {
					break
				}
				next, err := skipMvcc(data[start:])
				if err != nil {
					return 0, err
				}
				iNdEx = start + next
			}
			return iNdEx, nil
		case 4:
			return iNdEx, nil
		case 5:
			iNdEx += 4
			return iNdEx, nil
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
	}
	panic("unreachable")
}

var (
	ErrInvalidLengthMvcc = fmt.Errorf("proto: negative length found during unmarshaling")
)

func (m *MVCCValue) Size() (n int) {
	var l int
	_ = l
	n += 2
	if m.Value != nil {
		l = m.Value.Size()
		n += 1 + l + sovMvcc(uint64(l))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *MVCCMetadata) Size() (n int) {
	var l int
	_ = l
	if m.Txn != nil {
		l = m.Txn.Size()
		n += 1 + l + sovMvcc(uint64(l))
	}
	l = m.Timestamp.Size()
	n += 1 + l + sovMvcc(uint64(l))
	n += 2
	n += 1 + sovMvcc(uint64(m.KeyBytes))
	n += 1 + sovMvcc(uint64(m.ValBytes))
	if m.Value != nil {
		l = m.Value.Size()
		n += 1 + l + sovMvcc(uint64(l))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *MVCCStats) Size() (n int) {
	var l int
	_ = l
	n += 1 + sovMvcc(uint64(m.LiveBytes))
	n += 1 + sovMvcc(uint64(m.KeyBytes))
	n += 1 + sovMvcc(uint64(m.ValBytes))
	n += 1 + sovMvcc(uint64(m.IntentBytes))
	n += 1 + sovMvcc(uint64(m.LiveCount))
	n += 1 + sovMvcc(uint64(m.KeyCount))
	n += 1 + sovMvcc(uint64(m.ValCount))
	n += 1 + sovMvcc(uint64(m.IntentCount))
	n += 1 + sovMvcc(uint64(m.IntentAge))
	n += 1 + sovMvcc(uint64(m.GCBytesAge))
	n += 1 + sovMvcc(uint64(m.SysBytes))
	n += 1 + sovMvcc(uint64(m.SysCount))
	n += 2 + sovMvcc(uint64(m.LastUpdateNanos))
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func sovMvcc(x uint64) (n int) {
	for {
		n++
		x >>= 7
		if x == 0 {
			break
		}
	}
	return n
}
func sozMvcc(x uint64) (n int) {
	return sovMvcc(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *MVCCValue) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *MVCCValue) MarshalTo(data []byte) (n int, err error) {
	var i int
	_ = i
	var l int
	_ = l
	data[i] = 0x8
	i++
	if m.Deleted {
		data[i] = 1
	} else {
		data[i] = 0
	}
	i++
	if m.Value != nil {
		data[i] = 0x12
		i++
		i = encodeVarintMvcc(data, i, uint64(m.Value.Size()))
		n1, err := m.Value.MarshalTo(data[i:])
		if err != nil {
			return 0, err
		}
		i += n1
	}
	if m.XXX_unrecognized != nil {
		i += copy(data[i:], m.XXX_unrecognized)
	}
	return i, nil
}

func (m *MVCCMetadata) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *MVCCMetadata) MarshalTo(data []byte) (n int, err error) {
	var i int
	_ = i
	var l int
	_ = l
	if m.Txn != nil {
		data[i] = 0xa
		i++
		i = encodeVarintMvcc(data, i, uint64(m.Txn.Size()))
		n2, err := m.Txn.MarshalTo(data[i:])
		if err != nil {
			return 0, err
		}
		i += n2
	}
	data[i] = 0x12
	i++
	i = encodeVarintMvcc(data, i, uint64(m.Timestamp.Size()))
	n3, err := m.Timestamp.MarshalTo(data[i:])
	if err != nil {
		return 0, err
	}
	i += n3
	data[i] = 0x18
	i++
	if m.Deleted {
		data[i] = 1
	} else {
		data[i] = 0
	}
	i++
	data[i] = 0x20
	i++
	i = encodeVarintMvcc(data, i, uint64(m.KeyBytes))
	data[i] = 0x28
	i++
	i = encodeVarintMvcc(data, i, uint64(m.ValBytes))
	if m.Value != nil {
		data[i] = 0x32
		i++
		i = encodeVarintMvcc(data, i, uint64(m.Value.Size()))
		n4, err := m.Value.MarshalTo(data[i:])
		if err != nil {
			return 0, err
		}
		i += n4
	}
	if m.XXX_unrecognized != nil {
		i += copy(data[i:], m.XXX_unrecognized)
	}
	return i, nil
}

func (m *MVCCStats) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *MVCCStats) MarshalTo(data []byte) (n int, err error) {
	var i int
	_ = i
	var l int
	_ = l
	data[i] = 0x8
	i++
	i = encodeVarintMvcc(data, i, uint64(m.LiveBytes))
	data[i] = 0x10
	i++
	i = encodeVarintMvcc(data, i, uint64(m.KeyBytes))
	data[i] = 0x18
	i++
	i = encodeVarintMvcc(data, i, uint64(m.ValBytes))
	data[i] = 0x20
	i++
	i = encodeVarintMvcc(data, i, uint64(m.IntentBytes))
	data[i] = 0x28
	i++
	i = encodeVarintMvcc(data, i, uint64(m.LiveCount))
	data[i] = 0x30
	i++
	i = encodeVarintMvcc(data, i, uint64(m.KeyCount))
	data[i] = 0x38
	i++
	i = encodeVarintMvcc(data, i, uint64(m.ValCount))
	data[i] = 0x40
	i++
	i = encodeVarintMvcc(data, i, uint64(m.IntentCount))
	data[i] = 0x48
	i++
	i = encodeVarintMvcc(data, i, uint64(m.IntentAge))
	data[i] = 0x50
	i++
	i = encodeVarintMvcc(data, i, uint64(m.GCBytesAge))
	data[i] = 0x60
	i++
	i = encodeVarintMvcc(data, i, uint64(m.SysBytes))
	data[i] = 0x68
	i++
	i = encodeVarintMvcc(data, i, uint64(m.SysCount))
	data[i] = 0xf0
	i++
	data[i] = 0x1
	i++
	i = encodeVarintMvcc(data, i, uint64(m.LastUpdateNanos))
	if m.XXX_unrecognized != nil {
		i += copy(data[i:], m.XXX_unrecognized)
	}
	return i, nil
}

func encodeFixed64Mvcc(data []byte, offset int, v uint64) int {
	data[offset] = uint8(v)
	data[offset+1] = uint8(v >> 8)
	data[offset+2] = uint8(v >> 16)
	data[offset+3] = uint8(v >> 24)
	data[offset+4] = uint8(v >> 32)
	data[offset+5] = uint8(v >> 40)
	data[offset+6] = uint8(v >> 48)
	data[offset+7] = uint8(v >> 56)
	return offset + 8
}
func encodeFixed32Mvcc(data []byte, offset int, v uint32) int {
	data[offset] = uint8(v)
	data[offset+1] = uint8(v >> 8)
	data[offset+2] = uint8(v >> 16)
	data[offset+3] = uint8(v >> 24)
	return offset + 4
}
func encodeVarintMvcc(data []byte, offset int, v uint64) int {
	for v >= 1<<7 {
		data[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	data[offset] = uint8(v)
	return offset + 1
}