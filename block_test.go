package main

import (
	"fmt"
	"reflect"
	"testing"
)

func TestSerialize(t *testing.T) {
	// 测试用例1
	block := Block{}
	block_serialize := block.Serialize()
	rs := Deserialize(block_serialize)

	fmt.Println(rs)

	if !reflect.DeepEqual(*rs, block) {
		t.Errorf("TestSerialize failed, expected %v, got %v", block, rs)
	}
}
