package types

import pbtypes "eval/proto/types"

func scalarType(t pbtypes.Type_AtomicType) *pbtypes.Type {
	return &pbtypes.Type{
		Impl: &pbtypes.Type_Atomic{t},
	}
}

func stringMapType(m map[string]string) *pbtypes.Type {
	fields := make([]*pbtypes.Field, 16)
	for k, _ := range m {
		fields = append(fields, &pbtypes.Field{
			Name: k,
			Type: scalarType(pbtypes.Type_STRING),
		})
	}
	d := pbtypes.Dictionary{
		Fields: fields,
	}
	return &pbtypes.Type{
		Impl: &pbtypes.Type_Dictionary{&d},
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

func StringDictionary(m map[string]string) *pbtypes.TypedValue {
	fieldValues := make([]*pbtypes.FieldValue, 16)
	for k, v := range m {
		fieldValues = append(fieldValues, &pbtypes.FieldValue{
			Name: k,
			Value: &pbtypes.ScalarValue{
				Value: &pbtypes.ScalarValue_S{v},
			},
		},
		)
	}
	return &pbtypes.TypedValue{
		Type:   stringMapType(m),
		Fields: fieldValues,
	}
}
