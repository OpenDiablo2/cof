package cof

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gravestench/bitstream"
)

const (
	numBlocks             = 256
	maxRecordsPerBlock    = 67
	byteCountName         = 8
	byteCountSpeedPadding = 2
	numEvents             = 144
	speedDivisor          = 256
	speedBaseFPS          = 25
	milliseconds          = 1000
)

// AnimationData is a representation of the binary data from `data/global/AnimData.d2`
type AnimationData struct {
	hashTable
	blocks  [numBlocks]*block
	entries map[string][]*AnimationDataRecord
}

// GetRecordNames returns a slice of all record name strings
func (ad *AnimationData) GetRecordNames() []string {
	result := make([]string, 0)

	for name := range ad.entries {
		result = append(result, name)
	}

	return result
}

// GetRecord returns a single AnimationDataRecord with the given name string. If there is more
// than one record with the given name string, the last record entry will be returned.
func (ad *AnimationData) GetRecord(name string) *AnimationDataRecord {
	records := ad.GetRecords(name)
	numRecords := len(records)

	if numRecords < 1 {
		return nil
	}

	return records[numRecords-1]
}

// GetRecords returns all records that have the given name string. The AnimData.d2 files have
// multiple records with the same name, but other values in the record are different.
func (ad *AnimationData) GetRecords(name string) []*AnimationDataRecord {
	return ad.entries[name]
}

// Load loads the data into an AnimationData struct
func Load(data []byte) (*AnimationData, error) {
	reader := bitstream.NewReader().FromBytes(data...)
	animdata := &AnimationData{}
	hashIdx := 0

	animdata.entries = make(map[string][]*AnimationDataRecord)

	for blockIdx := range animdata.blocks {
		recordCount, err := reader.Next(32).Bits().AsUInt32()
		if err != nil {
			return nil, err
		}
		if recordCount > maxRecordsPerBlock {
			return nil, fmt.Errorf("more than %d records in block", maxRecordsPerBlock)
		}

		records := make([]*AnimationDataRecord, recordCount)

		for recordIdx := uint32(0); recordIdx < recordCount; recordIdx++ {
			nameBytes, err := reader.Next(byteCountName).Bytes().AsBytes()
			if err != nil {
				return nil, err
			}

			if nameBytes[byteCountName-1] != byte(0) {
				return nil, errors.New("animdata AnimationDataRecord name missing null terminator byte")
			}

			name := string(nameBytes)
			name = strings.ReplaceAll(name, string(byte(0)), "")

			animdata.hashTable[hashIdx] = hashName(name)

			frames, err := reader.Next(32).Bits().AsUInt32()
			if err != nil {
				return nil, err
			}

			speed, err := reader.Next(2).Bytes().AsUInt16()
			if err != nil {
				return nil, err
			}

			_, err = reader.Next(byteCountSpeedPadding).Bytes().AsBytes()
			if err != nil {
				return nil, err
			}

			events := make(map[int]FrameEvent)

			for eventIdx := 0; eventIdx < numEvents; eventIdx++ {
				b, err := reader.Next(1).Bytes().AsByte()
				if err != nil {
					return nil, err
				}

				event := FrameEvent(b)
				if event != EventNone {
					events[eventIdx] = event
				}
			}

			r := &AnimationDataRecord{
				name,
				frames,
				speed,
				events,
			}

			records[recordIdx] = r

			if _, found := animdata.entries[r.name]; !found {
				animdata.entries[r.name] = make([]*AnimationDataRecord, 0)
			}

			animdata.entries[r.name] = append(animdata.entries[r.name], r)
		}

		b := &block{
			recordCount,
			records,
		}

		animdata.blocks[blockIdx] = b
	}

	if reader.Position() != len(data) {
		return nil, errors.New("unable to parse animation data")
	}

	return animdata, nil
}
