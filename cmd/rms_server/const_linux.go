//go:build linux

package main

const (
	VarData     = "/tmp/rms"
	Storage     = "/storage/"
	Persist     = "/persist.xml"
	StorageRoot = VarData + Storage
	PersistFile = VarData + Persist
)
