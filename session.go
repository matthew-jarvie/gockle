package gockle

import (
	"fmt"

	"github.com/gocql/gocql"
	"github.com/maraino/go-mock"
)

func metadata(s *gocql.Session, keyspace string) (*gocql.KeyspaceMetadata, error) {
	var m, err = s.KeyspaceMetadata(keyspace)

	if err != nil {
		return nil, err
	}

	if m.DurableWrites && m.Name == keyspace && m.StrategyClass == "" && m.StrategyOptions == nil && m.Tables == nil {
		return nil, fmt.Errorf("gockle: keyspace %v invalid", keyspace)
	}

	return m, nil
}

// Session is a Cassandra connection. The Query methods run CQL queries. The
// Columns and Tables methods provide simple metadata.
type Session interface {
	// Batch returns a new Batch for the Session.
	Batch(kind BatchKind) Batch

	// Close closes the Session.
	Close()

	// Columns returns a map from column names to types for keyspace and table.
	// Schema changes during a session are not reflected; you must open a new
	// Session to observe them.
	Columns(keyspace, table string) (map[string]gocql.TypeInfo, error)

	// Exec executes the query for statement and arguments.
	Exec(statement string, arguments ...interface{}) error

	// Iterate executes the query for statement and arguments and returns an
	// Iterator for the results.
	Iterate(statement string, arguments ...interface{}) Iterator

	// Scan executes the query for statement and arguments and puts the first
	// result row in results.
	Scan(statement string, arguments, results []interface{}) error

	// ScanMap executes the query for statement and arguments and puts the first
	// result row in results.
	ScanMap(statement string, arguments []interface{}, results map[string]interface{}) error

	// ScanMapSlice executes the query for statement and arguments and returns all
	// the result rows.
	ScanMapSlice(statement string, arguments ...interface{}) ([]map[string]interface{}, error)

	// ScanMapTx executes the query for statement and arguments as a lightweight
	// transaction. If the query is not applied, it puts the current values for the
	// columns in results. It returns whether the query is applied.
	ScanMapTx(statement string, arguments []interface{}, results map[string]interface{}) (bool, error)

	// Tables returns the table names for keyspace. Schema changes during a session
	// are not reflected; you must open a new Session to observe them.
	Tables(keyspace string) ([]string, error)
}

var (
	_ Session = SessionMock{}
	_ Session = session{}
)

// NewSession returns a new Session for s.
func NewSession(s *gocql.Session) Session {
	return session{s: s}
}

// NewSimpleSession returns a new Session for hosts. It uses native protocol
// version 4.
func NewSimpleSession(hosts ...string) (Session, error) {
	var c = gocql.NewCluster(hosts...)

	c.ProtoVersion = 4

	var s, err = c.CreateSession()

	if err != nil {
		return nil, err
	}

	return session{s: s}, nil
}

// SessionMock is a mock Session. See github.com/maraino/go-mock.
type SessionMock struct {
	mock.Mock
}

// Batch implements Session.
func (m SessionMock) Batch(kind BatchKind) Batch {
	return m.Called(kind).Get(0).(Batch)
}

// Close implements Session.
func (m SessionMock) Close() {
	m.Called()
}

// Columns implements Session.
func (m SessionMock) Columns(keyspace, table string) (map[string]gocql.TypeInfo, error) {
	var r = m.Called(keyspace, table)

	return r.Get(0).(map[string]gocql.TypeInfo), r.Error(1)
}

// Exec implements Session.
func (m SessionMock) Exec(statement string, arguments ...interface{}) error {
	return m.Called(statement, arguments).Error(0)
}

// Iterate implements Session.
func (m SessionMock) Iterate(statement string, arguments ...interface{}) Iterator {
	return m.Called(statement, arguments).Get(0).(Iterator)
}

// Scan implements Session.
func (m SessionMock) Scan(statement string, arguments, results []interface{}) error {
	return m.Called(statement, arguments, results).Error(0)
}

// ScanMap implements Session.
func (m SessionMock) ScanMap(statement string, arguments []interface{}, results map[string]interface{}) error {
	return m.Called(statement, arguments, results).Error(0)
}

// ScanMapSlice implements Session.
func (m SessionMock) ScanMapSlice(statement string, arguments ...interface{}) ([]map[string]interface{}, error) {
	var r = m.Called(statement, arguments)

	return r.Get(0).([]map[string]interface{}), r.Error(1)
}

// ScanMapTx implements Session.
func (m SessionMock) ScanMapTx(statement string, arguments []interface{}, results map[string]interface{}) (bool, error) {
	var r = m.Called(statement, arguments, results)

	return r.Bool(0), r.Error(1)
}

// Tables implements Session.
func (m SessionMock) Tables(keyspace string) ([]string, error) {
	var r = m.Called(keyspace)

	return r.Get(0).([]string), r.Error(1)
}

type session struct {
	s *gocql.Session
}

func (s session) Batch(kind BatchKind) Batch {
	return batch{b: s.s.NewBatch(gocql.BatchType(kind)), s: s.s}
}

func (s session) Close() {
	s.s.Close()
}

func (s session) Columns(keyspace, table string) (map[string]gocql.TypeInfo, error) {
	var m, err = metadata(s.s, keyspace)

	if err != nil {
		return nil, err
	}

	var t, ok = m.Tables[table]

	if !ok {
		return nil, fmt.Errorf("gockle: table %v.%v invalid", keyspace, table)
	}

	var types = map[string]gocql.TypeInfo{}

	for n, c := range t.Columns {
		types[n] = c.Type
	}

	return types, nil
}

func (s session) Exec(statement string, arguments ...interface{}) error {
	return s.s.Query(statement, arguments...).Exec()
}

func (s session) Iterate(statement string, arguments ...interface{}) Iterator {
	return iterator{i: s.s.Query(statement, arguments...).Iter()}
}

func (s session) Scan(statement string, arguments, results []interface{}) error {
	return s.s.Query(statement, arguments...).Scan(results...)
}

func (s session) ScanMap(statement string, arguments []interface{}, results map[string]interface{}) error {
	return s.s.Query(statement, arguments...).MapScan(results)
}

func (s session) ScanMapSlice(statement string, arguments ...interface{}) ([]map[string]interface{}, error) {
	return s.s.Query(statement, arguments...).Iter().SliceMap()
}

func (s session) ScanMapTx(statement string, arguments []interface{}, results map[string]interface{}) (bool, error) {
	return s.s.Query(statement, arguments...).MapScanCAS(results)
}

func (s session) Tables(keyspace string) ([]string, error) {
	var m, err = metadata(s.s, keyspace)

	if err != nil {
		return nil, err
	}

	var ts []string

	for t := range m.Tables {
		ts = append(ts, t)
	}

	return ts, nil
}
