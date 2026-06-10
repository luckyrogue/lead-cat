package employeedir

import "testing"

const goodCSV = "full_name,email,department\n" +
	"Иванов Иван Иванович,I.Ivanov@Company.kz,Разработка\n" +
	"\n" +
	"  Петров Пётр  , p.petrov@company.kz ,  Маркетинг \n" +
	",noname@company.kz,Без имени\n" +
	"Без Почты,,Отдел\n" // empty email row is skipped

func TestParse_Good(t *testing.T) {
	recs, err := Parse([]byte(goodCSV))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(recs) != 3 {
		t.Fatalf("want 3 records, got %d: %+v", len(recs), recs)
	}

	if recs[0].Email != "i.ivanov@company.kz" || recs[0].FullName != "Иванов Иван Иванович" || recs[0].Dept != "Разработка" {
		t.Fatalf("rec0 wrong: %+v", recs[0])
	}
	if recs[1].FullName != "Петров Пётр" || recs[1].Email != "p.petrov@company.kz" || recs[1].Dept != "Маркетинг" {
		t.Fatalf("rec1 not trimmed/lowered: %+v", recs[1])
	}
}

func TestParse_CRLF(t *testing.T) {
	recs, err := Parse([]byte("full_name,email,department\r\nИ И,i@c.kz,Dev\r\n"))
	if err != nil || len(recs) != 1 || recs[0].Email != "i@c.kz" {
		t.Fatalf("CRLF parse failed: %v %+v", err, recs)
	}
}

func TestParse_BadHeader(t *testing.T) {
	if _, err := Parse([]byte("name,mail,dep\nA,a@c.kz,D\n")); err == nil {
		t.Fatal("expected header error")
	}
}

func TestParse_HeaderOnlyIsEmpty(t *testing.T) {
	recs, err := Parse([]byte("full_name,email,department\n"))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(recs) != 0 {
		t.Fatalf("want 0 records, got %d", len(recs))
	}
}

func TestParse_AllBlankIsEmpty(t *testing.T) {
	recs, err := Parse([]byte("\n\n\n"))
	if err != nil || len(recs) != 0 {
		t.Fatalf("want 0 records no error, got %d / %v", len(recs), err)
	}
}

func TestParse_EmptyInput(t *testing.T) {
	for _, data := range [][]byte{nil, {}} {
		recs, err := Parse(data)
		if err != nil || len(recs) != 0 {
			t.Fatalf("empty input: want 0 records no error, got %d / %v", len(recs), err)
		}
	}
}
