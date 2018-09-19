package grpchelper_test

import (
	"testing"

	types "github.com/gogo/protobuf/types"
	"github.com/src-d/lookout/util/grpchelper"
	"github.com/stretchr/testify/require"
)

func TestToStruct(t *testing.T) {
	require := require.New(t)

	inputMap := map[string]interface{}{
		"bool":   true,
		"int":    1,
		"string": "val",
		"float":  0.5,
		"nil":    nil,
		"array":  []string{"val1", "val2"},
		"map": map[string]int{
			"field1": 1,
		},
		"struct": struct {
			Val string
		}{Val: "val"},
	}

	expectedSt := &types.Struct{
		Fields: map[string]*types.Value{
			"bool": &types.Value{
				Kind: &types.Value_BoolValue{
					BoolValue: true,
				},
			},
			"int": &types.Value{
				Kind: &types.Value_NumberValue{
					NumberValue: 1,
				},
			},
			"string": &types.Value{
				Kind: &types.Value_StringValue{
					StringValue: "val",
				},
			},
			"float": &types.Value{
				Kind: &types.Value_NumberValue{
					NumberValue: 0.5,
				},
			},
			"nil": nil,
			"array": &types.Value{
				Kind: &types.Value_ListValue{
					ListValue: &types.ListValue{
						Values: []*types.Value{
							&types.Value{
								Kind: &types.Value_StringValue{
									StringValue: "val1",
								},
							},
							&types.Value{
								Kind: &types.Value_StringValue{
									StringValue: "val2",
								},
							},
						},
					},
				},
			},
			"map": &types.Value{
				Kind: &types.Value_StructValue{
					StructValue: &types.Struct{
						Fields: map[string]*types.Value{
							"field1": &types.Value{
								Kind: &types.Value_NumberValue{
									NumberValue: 1,
								},
							},
						},
					},
				},
			},
			"struct": &types.Value{
				Kind: &types.Value_StructValue{
					StructValue: &types.Struct{
						Fields: map[string]*types.Value{
							"Val": &types.Value{
								Kind: &types.Value_StringValue{
									StringValue: "val",
								},
							},
						},
					},
				},
			},
		},
	}

	st := grpchelper.ToPBStruct(inputMap)
	require.Equal(expectedSt, st)
}

func TestToStructMapInterfaceInterface(t *testing.T) {
	require := require.New(t)

	inputMap := map[string]interface{}{
		"map": map[interface{}]interface{}{
			"field1": "val",
		},
	}

	expectedSt := &types.Struct{
		Fields: map[string]*types.Value{
			"map": &types.Value{
				Kind: &types.Value_StructValue{
					StructValue: &types.Struct{
						Fields: map[string]*types.Value{
							"field1": &types.Value{
								Kind: &types.Value_StringValue{
									StringValue: "val",
								},
							},
						},
					},
				},
			},
		},
	}

	st := grpchelper.ToPBStruct(inputMap)
	require.Equal(expectedSt, st)
}

func TestToStructSliceInterfaceWithMap(t *testing.T) {
	require := require.New(t)

	inputMap := map[string]interface{}{
		"array": []interface{}{map[interface{}]interface{}{
			"field1": "val",
		}},
	}

	expectedSt := &types.Struct{
		Fields: map[string]*types.Value{
			"array": &types.Value{
				Kind: &types.Value_ListValue{
					ListValue: &types.ListValue{
						Values: []*types.Value{
							&types.Value{
								Kind: &types.Value_StructValue{
									StructValue: &types.Struct{
										Fields: map[string]*types.Value{
											"field1": &types.Value{
												Kind: &types.Value_StringValue{
													StringValue: "val",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	st := grpchelper.ToPBStruct(inputMap)
	require.Equal(expectedSt, st)
}
