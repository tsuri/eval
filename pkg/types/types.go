package types

import pbtypes "eval/proto/types"

func scalarType(t pbtypes.Type_AtomicType) *pbtypes.Type {
	return &pbtypes.Type{
		Impl: &pbtypes.Type_Atomic{t},
	}
}

func scalarValueString(s string) []*pbtypes.FieldValue {
	return []*pbtypes.FieldValue{
		{
			Value: &pbtypes.ScalarValue{
				Value: &pbtypes.ScalarValue_S{s},
			},
		},
	}
}

func StringScalar(s string) *pbtypes.TypedValue {
	return &pbtypes.TypedValue{
		Type:   scalarType(pbtypes.Type_STRING),
		Fields: scalarValueString(s),
	}
}

func PrettyPrint(v *pbtypes.TypedValue) {
}
