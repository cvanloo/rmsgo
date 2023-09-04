//go:build windows

package main

const (
	VarData     = "C:\\tmp\\rms"
	Storage     = "\\storage\\"
	Persist     = "\\persist.xml"
	StorageRoot = VarData + Storage
	PersistFile = VarData + Persist
)
