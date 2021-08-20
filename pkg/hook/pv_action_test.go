package hook

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPv_hook_action(t *testing.T) {
	tests := []struct {
		name        string
		hook        *PVHook
		obj         interface{}
		expectedObj interface{}
		actionType  HookActionType
	}{
		{
			name:        "when PV hook is nil, object should not be modified",
			hook:        nil,
			obj:         generateFakePvObj("name1", map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
			expectedObj: generateFakePvObj("name1", map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
			actionType:  HookActionAdd,
		},
		{
			name:        "when PV hook is configured to add metadata, object should be modified",
			hook:        buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
			obj:         generateFakePvObj("name2", nil, nil),
			expectedObj: generateFakePvObj("name2", map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
			actionType:  HookActionAdd,
		},
		{
			name:        "when PV hook is configured to remove metadata, object should be modified",
			hook:        buildPVHook(map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
			obj:         generateFakePvObj("name3", map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
			expectedObj: generateFakePvObj("name3", map[string]string{}, []string{}),
			actionType:  HookActionRemove,
		},
		{
			name:        "when PV hook is configured to remove non-existing metadata, object should not be modified",
			hook:        buildPVHook(map[string]string{"test.com/key": "val"}, []string{"test.com/finalizer"}),
			obj:         generateFakePvObj("name4", map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
			expectedObj: generateFakePvObj("name4", map[string]string{"test.io/key": "val"}, []string{"test.io/finalizer"}),
			actionType:  HookActionRemove,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.NotNil(t, test.obj, "object should not be nil")
			err := pv_hook_action(test.hook, test.actionType, test.obj)
			assert.Nil(t, err, "pv_hook_action returned error")
			assert.Equal(t, test.expectedObj, test.obj, "object should match")
		})
	}
}
