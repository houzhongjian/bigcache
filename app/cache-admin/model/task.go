package model

import (
	"github.com/jinzhu/gorm"
)

type TaskStatus int

type Task struct {
	Base
	SlotID     int        `json:"slot_id" gorm:"slot_id"`       //插槽id.
	MigrateIP  string     `json:"migrate_ip" gorm:"migrate_ip"` //插槽所在的ip.
	TargetIP   string     `json:"target_ip" gorm:"target_ip"`   //迁移到ip.
	Status     TaskStatus `json:"status" gorm:"stats"`          //任务状态.
	StatusName string     `json:"-" gorm:"-"`                   //任务状态.
	EndAt      string     `json:"end_at" gorm:"end_at"`         //任务结束时间.
}

func (task *Task) SwitchTaskStatus() {
	if task.Status == 0 {
		task.StatusName = "等待开始"
	}

	if task.Status == 1 {
		task.StatusName = "执行中"
	}

	if task.Status == 2 {
		task.StatusName = "执行结束"
	}
}

//TableName .
func (t *Task) TableName() string {
	return "task"
}

//GetTaskList 获取任务列表.
func (m *Model) GetTaskList() (list []*Task, err error) {
	task := Task{}
	err = m.db.Table(task.TableName()).Order("id desc").Find(&list).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return list, err
	}

	for _, item := range list {
		item.SwitchTaskStatus()
	}

	return list, nil
}

//CreateTask.
func (m *Model) CreateTask(task Task) error {
	return m.db.Model(&task).Create(&task).Error
}

//QueryTaskById 根据id查询任务.
func (m *Model) QueryTaskById(id int) (task Task, err error) {
	err = m.db.Table(task.TableName()).Find(&task, "id = ?", id).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return task, err
	}
	task.SwitchTaskStatus()
	return task, nil
}

//UpdateTaskStatusById 根据任务id更改任务状态.
func (m *Model) UpdateTaskStatusById(taskid int, status TaskStatus) error {
	task := Task{}
	return m.db.Table(task.TableName()).
		Where("id = ?", taskid).
		Update(map[string]interface{}{"status": status}).Error
}
