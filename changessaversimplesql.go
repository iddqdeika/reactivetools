package reactivetools

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/iddqdeika/rrr/helpful"
	"strconv"
	"strings"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
)

var ChangeValueConverters = struct {
	Default ChangeValueConverter
	Int     ChangeValueConverter
}{Default: defaultChangeValueConverter{}, Int: intChangeValueConverter{}}

func NewSimpleSqlChangesSaver(cfg helpful.Config, l helpful.Logger, c ChangeValueConverter) (ChangesProcessor, error) {
	if cfg == nil {
		return nil, fmt.Errorf("must be not-nil cfg")
	}
	if l == nil {
		return nil, fmt.Errorf("must be not-nil logger")
	}
	if c == nil {
		c = ChangeValueConverters.Default
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
	contentFlagColumn, err := cfg.GetString("data_column")
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlserver", connString)
	if err != nil {
		return nil, err
	}
	ti := targetInfo{
		tableName:            tableName,
		identifierColumnName: itemColumn,
		valueColumnName:      contentFlagColumn,
	}
	s := &sqlSaver{
		db: db,
		l:  l,
		p:  createDefaultProcessor(ti, c),
	}
	return s, nil
}

func NewCustomSqlChangesSaver(cfg helpful.Config, l helpful.Logger,
	pf SqlSaverProcessorFabric) (ChangesProcessor, error) {
	if cfg == nil {
		return nil, fmt.Errorf("must be not-nil cfg")
	}
	if l == nil {
		return nil, fmt.Errorf("must be not-nil logger")
	}
	if pf == nil {
		return nil, fmt.Errorf("must be not-nil SqlSaverProcessorFunc")
	}
	connString, err := cfg.GetString("conn_string")
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlserver", connString)
	if err != nil {
		return nil, err
	}

	p, err := pf.New(cfg)
	if err != nil {
		return nil, err
	}

	s := &sqlSaver{
		db: db,
		l:  l,
		p:  p,
	}
	return s, nil
}

type sqlSaver struct {
	db *sql.DB
	l  helpful.Logger

	p SqlSaverProcessor
}

func (s *sqlSaver) Process(event ChangeEvent) error {
	return s.p.Process(s.db, event, s.l)
}

func createDefaultProcessor(i targetInfo,
	c ChangeValueConverter) SqlSaverProcessor {
	return &defaultProcessor{
		i: i,
		c: c,
	}
}

type defaultProcessor struct {
	i targetInfo
	c ChangeValueConverter
}

func (p *defaultProcessor) Process(db *sql.DB, event ChangeEvent, l helpful.Logger) error {
	query := fmt.Sprintf(`if not exists 
(select * from %v where %v = @p1)
insert into %v (%v, %v) values (@p1,@p2)
else
update %v set %v = @p2 where %v = @p1
`, p.i.tableName, p.i.identifierColumnName, p.i.tableName, p.i.identifierColumnName,
		p.i.valueColumnName, p.i.tableName, p.i.valueColumnName, p.i.identifierColumnName)

	data, err := p.c.Convert(event)
	if err != nil {
		return fmt.Errorf("cant convert data: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	_, err = db.ExecContext(ctx, query, event.ObjectIdentifier(), data)
	if err != nil {
		return err
	}
	l.Infof("value (%v) for entity(%v): %v saved", event.Data(), event.ObjectType(), event.ObjectIdentifier())
	return nil
}

type targetInfo struct {
	tableName            string
	identifierColumnName string
	valueColumnName      string
}

type defaultChangeValueConverter struct {
}

func (d defaultChangeValueConverter) Convert(event ChangeEvent) (interface{}, error) {
	return strings.Replace(event.Data(), ",", ";", -1), nil
}

type intChangeValueConverter struct {
}

func (i intChangeValueConverter) Convert(event ChangeEvent) (interface{}, error) {
	return strconv.Atoi(event.Data())
}
