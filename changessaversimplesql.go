package reactivetools

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/iddqdeika/rrr/helpful"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
)

func NewSimpleSqlChangesSaver(cfg helpful.Config, l helpful.Logger) (ChangesProcessor, error) {
	if cfg == nil {
		return nil, fmt.Errorf("must be not-nil cfg")
	}
	if l == nil {
		return nil, fmt.Errorf("must be not-nil logger")
	}

	connString, err := cfg.GetString("conn_string")
	if err != nil {
		return nil, err
	}
	tableName, err := cfg.GetString("item_flags_table")
	if err != nil {
		return nil, err
	}
	itemColumn, err := cfg.GetString("item_column")
	if err != nil {
		return nil, err
	}
	contentFlagColumn, err := cfg.GetString("content_flags_column")
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlserver", connString)
	if err != nil {
		return nil, err
	}
	s := &sqlSaver{
		db:                   db,
		l:                    l,
		tableName:            tableName,
		identifierColumnName: itemColumn,
		valueColumnName:      contentFlagColumn,
	}
	return s, nil
}

type sqlSaver struct {
	db *sql.DB
	l  helpful.Logger

	tableName            string
	identifierColumnName string
	valueColumnName      string
}

func (s *sqlSaver) Process(event ChangeEvent) error {
	query := fmt.Sprintf(`if not exists 
(select * from %v where %v = @p1)
insert into %v (%v, %v) values (@p1,@p2)
else
update %v set %v = @p2 where %v = @p1
`, s.tableName, s.identifierColumnName, s.tableName, s.identifierColumnName,
		s.valueColumnName, s.tableName, s.valueColumnName, s.identifierColumnName)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	_, err := s.db.ExecContext(ctx, query, event.ObjectIdentifier(), event.Data())
	if err != nil {
		return err
	}
	s.l.Infof("value (%v) for entity(%v): %v saved", event.Data(), event.ObjectType(), event.ObjectIdentifier())
	return nil
}
