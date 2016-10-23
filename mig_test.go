package mig

import (
	"log"
	"os/exec"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

//todo: also test with mysql

func Test(t *testing.T) {
	exec.Command("dropdb", "testPostgres").Run()
	output, err := exec.Command("createdb", "testPostgres").CombinedOutput()
	defer exec.Command("dropdb", "testPostgres").Run()
	if err != nil {
		log.Printf("couldn't create postgres db: %v, %s\n", err, output)
	}

	pg, err := sqlx.Connect("postgres", "postgres://testuser:testpassword@localhost/testPostgres")
	if err != nil {
		t.Fatalf("couldn't connect to postgres test db: %v\n", err)
	}

	t.Run("revert", func(t *testing.T) {
		testRevert(t, pg)
	})

	t.Run("prereq", func(t *testing.T) {
		testPrereq(t, pg)
	})

	t.Run("whitespace", func(t *testing.T) {
		testWhitespace(t, pg)
	})
}

func testRevert(t *testing.T, db *sqlx.DB) {
	Register([]Step{
		{
			Migrate: `create table survive(val int)`,
		},
		{
			Migrate: `
			  create table test_user (
					name TEXT,
					food TEXT
				)
			`,
			Revert: `drop table test_user`,
		},
	}, "TestRevert-1")

	err := Run(db, "TestRevert-1")
	if err != nil {
		t.Fatalf(": %v\n", err)
	}

	stmt := `insert into survive (val) values (42)`
	if _, err = db.Exec(stmt); err != nil {
		t.Fatalf("couldn't insert: %v\n", err)
	}

	stmt = `insert into test_user (name, food) values ('Jonathan', 'crab'), ('Sarah', 'ice cream')`
	if _, err = db.Exec(stmt); err != nil {
		t.Fatalf("couldn't insert: %v\n", err)
	}

	var result1 []struct {
		Name string `db:"name"`
		Food string `db:"food"`
	}

	db.Select(&result1, "select * from test_user")
	if len(result1) != 2 {
		t.Fatalf(`len(result) != 2, len(result) == "%v"`, len(result1))
	}

	Register([]Step{
		{
			Migrate: `create table survive(val int)`,
		},
		{
			Migrate: `
				create table test_user (
					name TEXT,
					tv   TEXT
				)
			`,
		},
	}, "TestRevert-2")
	err = Run(db, "TestRevert-2")
	if err != nil {
		t.Fatalf(": %v\n", err)
	}

	stmt = `insert into test_user (name, tv) values ('Jonathan', 'Rick and Morty'), ('Sarah', 'The Office')`
	if _, err = db.Exec(stmt); err != nil {
		t.Fatalf("couldn't insert: %v\n", err)
	}

	var result2 []struct {
		Name string `db:"name"`
		Tv   string `db:"tv"`
	}
	db.Select(&result2, "select * from test_user")
	if len(result2) != 2 {
		t.Fatalf(`len(result2) != 2, len(result2) == "%v"`, len(result2))
	}

	var surviver struct {
		Val int
	}
	err = db.Get(&surviver, `select val from survive limit 1`)
	if err != nil {
		t.Fatalf("table 'survive' didn't survive as expected: %v\n", err)
	}
	if surviver.Val != 42 {
		t.Fatalf(`surviver.Val != 42, surviver.Val == "%v"`, surviver.Val)
	}
}

func testPrereq(t *testing.T, db *sqlx.DB) {
	Register([]Step{
		{
			Prereq: `
			  select 1 from test_prereq
			`,
			Migrate: `
			  alter table test_prereq add column food text
			`,
		},
	}, "TestPrereq")

	Register([]Step{
		{
			Migrate: `
			  create table test_prereq()
			`,
		},
	}, "TestPrereq")

	err := Run(db, "TestPrereq")
	if err != nil {
		t.Fatalf("couldn't run migrations: %v\n", err)
	}

	_, err = db.Exec("insert into test_prereq(food) values ('nachos'), ('burritos')")
	if err != nil {
		t.Fatalf("couldn't run migration: %v\n", err)
	}

	var result []struct {
		Food string
	}
	err = db.Select(&result, "select * from test_prereq")
	if err != nil {
		t.Fatalf(": %v\n", err)
	}

	if len(result) != 2 {
		t.Fatalf(`len(result) != 2, len(result) == "%v"`, len(result))
	}
}

func testWhitespace(t *testing.T, db *sqlx.DB) {
	Register([]Step{
		{
			Revert: `drop table test_whitespace`,
			Migrate: `
			  --comments shouldn't affect things...
				create table test_whitespace(
					survive int
				)
			`,
		},
	}, "TestWhitespace-1")

	err := Run(db, "TestWhitespace-1")
	if err != nil {
		t.Fatalf(": %v\n", err)
	}

	//insert a value which is expected to survive the migration below
	_, err = db.Exec("insert into test_whitespace values (42)")
	if err != nil {
		t.Fatalf("couldn't insert: %v\n", err)
	}

	//this is the same migration, except for whitespace differences
	Register([]Step{
		{
			Revert: `drop table test_whitespace`,
			Migrate: strings.Join([]string{
				"create table test_whitespace(",
				"survive int",
				")",
			}, "\n"),
		},
	}, "TestWhitespace-2")

	err = Run(db, "TestWhitespace-2")
	if err != nil {
		t.Fatalf(": %v\n", err)
	}

	//check if the value survived as expected
	var result struct {
		Survive int
	}
	err = db.Get(&result, "select * from test_whitespace")
	if err != nil {
		t.Fatalf("couldn't select: %v\n", err)
	}

	if result.Survive != 42 {
		t.Fatalf(`result.Survive != 42, result.Survive == "%v"`, result.Survive)
	}
}