package pgtrigger

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/lib/pq"
	"time"
)

// Msg 触发器消息格式
type Msg struct {
	Schema string `json:"schema"`
	Table  string `json:"table"`
	Action string `json:"action"`
	// old_data 和 new_data 为 map[string]interface{} 结构的json字符串，
	// 如果定义了表结构，可以直接反序列化为表对象
	OldData string `json:"old_data"`
	NewData string `json:"new_data"`
}

// Trigger
// 触发器的文档：http://postgres.cn/docs/12/trigger-definition.html
// PL/pgSQL 编程文档：http://postgres.cn/docs/12/plpgsql.html
// 触发器函数文档：http://postgres.cn/docs/12/plpgsql-trigger.html
type Trigger struct {
	ConnStr string
	Schema  string
	Table   string

	listener  *pq.Listener
	callbacks []func(msg Msg)

	// 中止信号
	// 999: 手动停止
	// 888: 连接异常停止
	abortSignal chan int

	// 运行状态
	// -2: 监听连接异常
	// -1: 回调函数执行异常
	//  0: 初始状态，未启动
	//  1: 监听器已启动
	//  2: 正在监听并执行，正常运行状态
	//  3: 手动停止
	state int
	err   error
}

func (t *Trigger) channel() string {
	return fmt.Sprintf("%s_%s_channel", t.Schema, t.Table)
}

func (t *Trigger) name() string {
	return fmt.Sprintf("%s_%s_trigger", t.Schema, t.Table)
}

// Clear 清除表触发器
func (t *Trigger) Clear() {
	trigger := t.name()
	fullTable := fmt.Sprintf("%s.%s", t.Schema, t.Table)

	triggerSql := fmt.Sprintf(`
-- 删除旧触发器
DROP TRIGGER IF EXISTS %s ON %s ;
`, trigger, fullTable)

	db, err := sql.Open("postgres", t.ConnStr)
	if err != nil {
		err = fmt.Errorf("connect postgres error:%v", err)
		return
	}

	defer db.Close()

	_, err = db.Exec(triggerSql)
	return
}

// Set 设置表触发器，如果存在会删除新建
func (t *Trigger) Set() (err error) {
	channel := t.channel()
	trigger := t.name()
	fullTable := fmt.Sprintf("%s.%s", t.Schema, t.Table)

	triggerSql := fmt.Sprintf(`
-- 触发器函数;
CREATE OR REPLACE FUNCTION notify_change() RETURNS TRIGGER AS $$
DECLARE
  notification json;
  v_old_data TEXT;
  v_new_data TEXT;
BEGIN
  v_old_data := '';
  v_new_data := '';
  IF (TG_OP = 'UPDATE') THEN
      v_old_data = row_to_json(OLD);
      v_new_data = row_to_json(NEW);
  ELSIF (TG_OP = 'DELETE') THEN
      v_old_data = row_to_json(OLD);
  ELSIF (TG_OP = 'INSERT') THEN
      v_new_data = row_to_json(NEW);
  END IF;
  notification = json_build_object(
                          'schema',TG_TABLE_SCHEMA,
                          'table',TG_TABLE_NAME,
                          'new_data', v_new_data,
                          'old_data', v_old_data,
						  'action', TG_OP);
  PERFORM pg_notify('%s',notification::text);
  RETURN NULL;
END; $$ LANGUAGE plpgsql;

-- 删除旧触发器
DROP TRIGGER IF EXISTS %s ON %s ;

-- 为用户表创建行级触发器，监听INSERT UPDATE DELETE 操作。
CREATE TRIGGER %s AFTER INSERT OR UPDATE OR DELETE ON %s
FOR EACH ROW EXECUTE PROCEDURE notify_change();
`, channel, trigger, fullTable, trigger, fullTable)

	db, err := sql.Open("postgres", t.ConnStr)
	if err != nil {
		err = fmt.Errorf("connect postgres error:%v", err)
		return
	}

	defer db.Close()

	_, err = db.Exec(triggerSql)
	return
}

// Listen 监听表数据改动
func (t *Trigger) Listen(callbacks []func(msg Msg)) (err error) {
	if t.ListenerAlive() {
		err = fmt.Errorf("already listen")
		return
	}

	listener := pq.NewListener(t.ConnStr, time.Millisecond*500, time.Millisecond*5000, func(event pq.ListenerEventType, err error) {
		if event == pq.ListenerEventDisconnected || event == pq.ListenerEventConnectionAttemptFailed {
			t.err = fmt.Errorf("listener disconnected: %v", err)
			t.abortSignal <- 888
		}
	})

	err = listener.Listen(t.channel())
	if err != nil {
		err = fmt.Errorf("listen error: %v, check if the channel is present", err)
		return
	}

	t.listener = listener
	t.callbacks = callbacks
	t.abortSignal = make(chan int)
	t.state = 1

	go t.do()

	return
}

// do 执行回调函数
func (t *Trigger) do() {
	defer func() {
		if err := recover(); err != nil {
			t.state = -1
			t.err = fmt.Errorf("do error: %v", err)
		}
	}()

	t.state = 2
	for {
		select {
		case signal := <-t.abortSignal:
			switch signal {
			case 888:
				t.state = -2
			case 999:
				t.state = 3
			}
			return
		case i := <-t.listener.Notify:
			msg := new(Msg)
			_ = json.Unmarshal([]byte(i.Extra), msg)
			for _, cb := range t.callbacks {
				cb(*msg)
			}
		}
	}
}

func (t *Trigger) ListenerAlive() bool {
	if t.listener == nil {
		return false
	} else {
		return t.listener.Ping() == nil
	}
}

func (t *Trigger) State() int {
	return t.state
}

func (t *Trigger) Err() error {
	return t.err
}

// Stop 停止监听，只能调用一次
func (t *Trigger) Stop() {
	if t.state != 2 {
		return
	}

	t.abortSignal <- 999
	_ = t.listener.Close()

	for i := 0; i < 100; i++ {
		if t.state == 3 {
			return
		}
		time.Sleep(time.Millisecond * 100)
	}

	close(t.abortSignal)
}
