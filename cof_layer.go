package cof

// CofLayer is a structure that represents a single layer in a COF file.
type CofLayer struct {
	Type        CompositeType
	Shadow      byte
	Selectable  bool
	Transparent bool
	DrawEffect  DrawEffect
	WeaponClass WeaponClass
}
