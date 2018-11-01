# MP2 - Membership - Group 57

## Group Members

Jiazheng Li (jl46) & Yabo Li (yaboli2)

## Environment

Go 1.9.4

## Communication Framework & Protocol

RPC & UDP

## Marshalling Library

encoding/json

## Usage

```bash
# build
go build swim.go # generate executeble
# or just run
go run swim.go

The program would asynchronously listen and ping, by UDP.

All status update about the current group would be displayed on terminal, as well as written in log files.

Type "show" to show the full current membership list on the local VM.

Type "leave" to let one VM voluntarily leave the group and inform other processes.