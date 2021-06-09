package cof

import (
	"strings"

	"github.com/gravestench/bitstream"
)

const (
	numUnknownHeaderBytes = 21
	numUnknownBodyBytes   = 3
	numHeaderBytes        = 4 + numUnknownHeaderBytes
	numLayerBytes         = 9
)

const (
	headerNumLayers = iota
	headerFramesPerDir
	headerNumDirs
	headerSpeed = numHeaderBytes - 1
)

const (
	layerType = iota
	layerShadow
	layerSelectable
	layerTransparent
	layerDrawEffect
	layerWeaponClass
)

const (
	badCharacter = string(byte(0))
)

// FrameEvent represents a single frame of animation.
type FrameEvent int

// FrameEvent types
const (
	EventNone FrameEvent = iota
	EventAttack
	EventMissile
	EventSound
	EventSkill
)


// New creates a new COF
func New() *COF {
	return &COF{
		unknownHeaderBytes: make([]byte, numUnknownHeaderBytes),
		unknownBodyBytes:   make([]byte, numUnknownBodyBytes),
		NumberOfDirections: 0,
		FramesPerDirection: 0,
		NumberOfLayers:     0,
		Speed:              0,
		CofLayers:          make([]CofLayer, 0),
		CompositeLayers:    make(map[CompositeType]int),
		AnimationFrames:    make([]FrameEvent, 0),
		Priority:           make([][][]CompositeType, 0),
	}
}

// Marshal a COF to a new byte slice
func Marshal(c *COF) []byte {
	return c.Marshal()
}

// Unmarshal a byte slice to a new COF
func Unmarshal(data []byte) (*COF, error) {
	c := New()
	err := c.Unmarshal(data)

	return c, err
}

// COF is a structure that represents a COF file.
type COF struct {
	// unknown bytes for header
	unknownHeaderBytes []byte
	// unknown bytes (first "body's" bytes)
	unknownBodyBytes   []byte
	NumberOfDirections int
	FramesPerDirection int
	NumberOfLayers     int
	Speed              int
	CofLayers          []CofLayer
	CompositeLayers    map[CompositeType]int
	AnimationFrames    []FrameEvent
	Priority           [][][]CompositeType
}

// Unmarshal a byte slice to this COF
func (c *COF) Unmarshal(fileData []byte) error {
	var err error

	stream := bitstream.NewReader().FromBytes(fileData...)

	headerBytes, err := stream.Next(numHeaderBytes).Bytes().AsBytes()
	if err != nil {
		return err
	}

	c.loadHeader(headerBytes)

	c.unknownBodyBytes, err = stream.Next(numUnknownBodyBytes).Bytes().AsBytes()
	if err != nil {
		return err
	}

	c.CofLayers = make([]CofLayer, c.NumberOfLayers)
	c.CompositeLayers = make(map[CompositeType]int)

	err = c.loadCOFLayers(stream)
	if err != nil {
		return err
	}

	animationFramesData, err := stream.Next(c.FramesPerDirection).Bytes().AsBytes()
	if err != nil {
		return err
	}

	c.loadAnimationFrames(animationFramesData)

	priorityLen := c.FramesPerDirection * c.NumberOfDirections * c.NumberOfLayers
	c.Priority = make([][][]CompositeType, c.NumberOfDirections)

	priorityBytes, err := stream.Next(priorityLen).Bytes().AsBytes()
	if err != nil {
		return err
	}

	c.loadPriority(priorityBytes)

	return nil
}

func (c *COF) loadHeader(b []byte) {
	c.NumberOfLayers = int(b[headerNumLayers])
	c.FramesPerDirection = int(b[headerFramesPerDir])
	c.NumberOfDirections = int(b[headerNumDirs])
	c.unknownHeaderBytes = b[headerNumDirs+1 : headerSpeed]
	c.Speed = int(b[headerSpeed])
}

func (c *COF) loadCOFLayers(s *bitstream.Reader) error {
	for i := 0; i < c.NumberOfLayers; i++ {
		layer := CofLayer{}

		b, err := s.Next(numLayerBytes).Bytes().AsBytes()
		if err != nil {
			return err
		}

		layer.Type = CompositeType(b[layerType])
		layer.Shadow = b[layerShadow]
		layer.Selectable = b[layerSelectable] > 0
		layer.Transparent = b[layerTransparent] > 0
		layer.DrawEffect = DrawEffect(b[layerDrawEffect])

		layer.WeaponClass = WeaponClassFromString(strings.TrimSpace(strings.ReplaceAll(
			string(b[layerWeaponClass:]), badCharacter, "")))

		c.CofLayers[i] = layer
		c.CompositeLayers[layer.Type] = i
	}

	return nil
}

func (c *COF) loadAnimationFrames(b []byte) {
	c.AnimationFrames = make([]FrameEvent, c.FramesPerDirection)

	for i := range b {
		c.AnimationFrames[i] = FrameEvent(b[i])
	}
}

func (c *COF) loadPriority(priorityBytes []byte) {
	priorityIndex := 0

	for direction := 0; direction < c.NumberOfDirections; direction++ {
		c.Priority[direction] = make([][]CompositeType, c.FramesPerDirection)
		for frame := 0; frame < c.FramesPerDirection; frame++ {
			c.Priority[direction][frame] = make([]CompositeType, c.NumberOfLayers)
			for i := 0; i < c.NumberOfLayers; i++ {
				c.Priority[direction][frame][i] = CompositeType(priorityBytes[priorityIndex])
				priorityIndex++
			}
		}
	}
}

// Marshal this COF to a byte slice
func (c *COF) Marshal() []byte {
	sw := &bitstream.Writer{}

	sw.WriteByte(byte(c.NumberOfLayers))
	sw.WriteByte(byte(c.FramesPerDirection))
	sw.WriteByte(byte(c.NumberOfDirections))
	sw.WriteBytes(c.unknownHeaderBytes)
	sw.WriteByte(byte(c.Speed))
	sw.WriteBytes(c.unknownBodyBytes)

	for i := range c.CofLayers {
		sw.WriteByte(byte(c.CofLayers[i].Type))
		sw.WriteByte(c.CofLayers[i].Shadow)

		if c.CofLayers[i].Selectable {
			sw.WriteByte(byte(1))
		} else {
			sw.WriteByte(byte(0))
		}

		if c.CofLayers[i].Transparent {
			sw.WriteByte(byte(1))
		} else {
			sw.WriteByte(byte(0))
		}

		sw.WriteByte(byte(c.CofLayers[i].DrawEffect))

		const (
			maxCodeLength = 3 // we assume item codes to look like 'hax' or 'kit'
			terminator    = 0
		)

		weaponCode := c.CofLayers[i].WeaponClass.String()

		for idx, letter := range weaponCode {
			if idx > maxCodeLength {
				break
			}

			sw.WriteByte(byte(letter))
		}

		sw.WriteByte(terminator)
	}

	for _, i := range c.AnimationFrames {
		sw.WriteByte(byte(i))
	}

	for direction := 0; direction < c.NumberOfDirections; direction++ {
		for frame := 0; frame < c.FramesPerDirection; frame++ {
			for i := 0; i < c.NumberOfLayers; i++ {
				sw.WriteByte(byte(c.Priority[direction][frame][i]))
			}
		}
	}

	return sw.Bytes()
}
