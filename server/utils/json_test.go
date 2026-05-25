package utils

import "testing"

func TestGetJSONKeys(t *testing.T) {
	var jsonStr = `
	{
		"Name": "test",
		"TableName": "test",
		"TemplateID": "test",
		"TemplateInfo": "test",
		"Limit": 0
}`
	keys, err := GetJSONKeys(jsonStr)
	if err != nil {
		t.Fatalf("GetJSONKeys failed: %v", err)
	}
	if len(keys) != 5 {
		t.Fatalf("expected 5 keys, got %d: %v", len(keys), keys)
	}
	if keys[0] != "Name" {
		t.Fatalf("expected key[0] Name, got %q", keys[0])
	}
	if keys[1] != "TableName" {
		t.Fatalf("expected key[1] TableName, got %q", keys[1])
	}
	if keys[2] != "TemplateID" {
		t.Fatalf("expected key[2] TemplateID, got %q", keys[2])
	}
	if keys[3] != "TemplateInfo" {
		t.Fatalf("expected key[3] TemplateInfo, got %q", keys[3])
	}
	if keys[4] != "Limit" {
		t.Fatalf("expected key[4] Limit, got %q", keys[4])
	}
}
