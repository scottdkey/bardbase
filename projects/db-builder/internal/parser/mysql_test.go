package parser

import (
	"strings"
	"testing"
)

func TestExtractStatements_SkipsComments(t *testing.T) {
	sql := `-- This is a comment
INSERT INTO ` + "`Works`" + ` VALUES ('hamlet');
/* Block comment */
INSERT INTO ` + "`Characters`" + ` VALUES ('ophelia');`

	stmts := ExtractStatements(sql)
	if len(stmts) != 2 {
		t.Fatalf("expected 2 statements, got %d: %v", len(stmts), stmts)
	}
}

func TestExtractStatements_HandlesQuotedSemicolons(t *testing.T) {
	sql := "INSERT INTO `Characters` VALUES ('Bassianus', 'brother to Saturninus;', 'titus');"
	stmts := ExtractStatements(sql)

	if len(stmts) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(stmts))
	}
	if !strings.Contains(stmts[0], "brother to Saturninus;") {
		t.Error("semicolon inside quotes should be preserved in the statement")
	}
}

func TestExtractStatements_MultipleStatements(t *testing.T) {
	sql := `INSERT INTO a VALUES (1);
INSERT INTO b VALUES (2);
INSERT INTO c VALUES (3);`

	stmts := ExtractStatements(sql)
	if len(stmts) != 3 {
		t.Fatalf("expected 3 statements, got %d", len(stmts))
	}
}

func TestParseMySQLValues_SingleRow(t *testing.T) {
	stmt := "INSERT INTO `Works` VALUES ('hamlet', 'Hamlet', 'The Tragedy of Hamlet', 1601)"
	rows := ParseMySQLValues(stmt)

	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if len(rows[0]) != 4 {
		t.Fatalf("expected 4 fields, got %d", len(rows[0]))
	}
	if ValStr(rows[0][0]) != "hamlet" {
		t.Errorf("field 0: expected 'hamlet', got %q", ValStr(rows[0][0]))
	}
	if ValStr(rows[0][3]) != "1601" {
		t.Errorf("field 3: expected '1601', got %q", ValStr(rows[0][3]))
	}
}

func TestParseMySQLValues_MultipleRows(t *testing.T) {
	stmt := "INSERT INTO `Works` VALUES ('hamlet', 'Hamlet'), ('othello', 'Othello');"
	rows := ParseMySQLValues(stmt)

	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if ValStr(rows[0][0]) != "hamlet" {
		t.Errorf("row 0 field 0: expected 'hamlet', got %q", ValStr(rows[0][0]))
	}
	if ValStr(rows[1][0]) != "othello" {
		t.Errorf("row 1 field 0: expected 'othello', got %q", ValStr(rows[1][0]))
	}
}

func TestParseMySQLValues_NullHandling(t *testing.T) {
	stmt := "INSERT INTO `Works` VALUES ('hamlet', NULL, 'tragedy')"
	rows := ParseMySQLValues(stmt)

	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0][1] != nil {
		t.Errorf("field 1: expected nil (NULL), got %q", ValStr(rows[0][1]))
	}
}

func TestParseMySQLValues_EscapedQuotes(t *testing.T) {
	stmt := `INSERT INTO ` + "`Characters`" + ` VALUES ('it''s a test', 'backslash\'s too')`
	rows := ParseMySQLValues(stmt)

	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if ValStr(rows[0][0]) != "it's a test" {
		t.Errorf("field 0: expected \"it's a test\", got %q", ValStr(rows[0][0]))
	}
}

func TestParseMySQLValues_NoValues(t *testing.T) {
	stmt := "SELECT * FROM works"
	rows := ParseMySQLValues(stmt)

	if rows != nil {
		t.Errorf("expected nil for non-INSERT statement, got %d rows", len(rows))
	}
}

func TestGetInsertTable_Backticked(t *testing.T) {
	stmt := "INSERT INTO `Characters` VALUES (1, 'Hamlet')"
	table := GetInsertTable(stmt)
	if table != "Characters" {
		t.Errorf("expected 'Characters', got %q", table)
	}
}

func TestGetInsertTable_Plain(t *testing.T) {
	stmt := "INSERT INTO works VALUES (1, 'Hamlet')"
	table := GetInsertTable(stmt)
	if table != "works" {
		t.Errorf("expected 'works', got %q", table)
	}
}

func TestGetInsertTable_NotInsert(t *testing.T) {
	stmt := "SELECT * FROM works"
	table := GetInsertTable(stmt)
	if table != "" {
		t.Errorf("expected empty string, got %q", table)
	}
}

func TestValStr_Nil(t *testing.T) {
	if got := ValStr(nil); got != "" {
		t.Errorf("ValStr(nil) = %q, want empty string", got)
	}
}

func TestValStr_NonNil(t *testing.T) {
	s := "hello"
	if got := ValStr(&s); got != "hello" {
		t.Errorf("ValStr(&'hello') = %q, want 'hello'", got)
	}
}
