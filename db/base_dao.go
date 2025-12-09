package db

import (
	"context"
	"errors"

	"github.com/SUPERDBFMP/gorm-plus-enhanced/gplus"
	"gorm.io/gorm"
)

type BaseDao[T any] struct {
}

type BaseDaoWithComparable[T any, V gplus.Comparable] struct {
	*BaseDao[T]
}

type BaseDaoGeneric[T any, R any] struct {
	*BaseDao[T]
}

type BaseDaoStreamingGeneric[T any, R any, V gplus.Comparable] struct {
	*BaseDao[T]
}

func NewBaseDao[T any]() *BaseDao[T] {
	return &BaseDao[T]{}
}

// NewQueryCond 创建查询条件
func (b *BaseDao[T]) NewQueryCond() (*gplus.QueryCond[T], *T) {
	return gplus.NewQuery[T]()
}

func NewBaseDaoWithComparable[T any, V gplus.Comparable]() *BaseDaoWithComparable[T, V] {
	return &BaseDaoWithComparable[T, V]{BaseDao: NewBaseDao[T]()}
}

func NewBaseDaoGeneric[T any, R any]() *BaseDaoGeneric[T, R] {
	return &BaseDaoGeneric[T, R]{BaseDao: NewBaseDao[T]()}
}

func NewBaseDaoStreamingGeneric[T any, R any, V gplus.Comparable]() *BaseDaoStreamingGeneric[T, R, V] {
	return &BaseDaoStreamingGeneric[T, R, V]{BaseDao: NewBaseDao[T]()}
}

//---------------------------------------------------查询------------------------------------------------------//

// SelectById 根据 ID 查询单条记录
func (b *BaseDao[T]) SelectById(ctx context.Context, id any, opts ...gplus.OptionFunc) (*T, error) {
	result, db := gplus.SelectById[T](ctx, id, opts...)
	if db.Error != nil {
		if errors.Is(db.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		} else {
			return nil, db.Error
		}
	}
	return result, nil
}

// SelectByIds 根据 ID 查询多条记录
func (b *BaseDao[T]) SelectByIds(ctx context.Context, ids any, opts ...gplus.OptionFunc) ([]*T, error) {
	result, db := gplus.SelectByIds[T](ctx, ids, opts...)
	return result, db.Error
}

// SelectOne 根据条件查询单条记录
func (b *BaseDao[T]) SelectOne(ctx context.Context, q *gplus.QueryCond[T], opts ...gplus.OptionFunc) (*T, error) {
	result, db := gplus.SelectOne[T](ctx, q, opts...)
	if db.Error != nil {
		if errors.Is(db.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		} else {
			return nil, db.Error
		}
	}
	return result, nil
}

// SelectList 根据条件查询多条记录
func (b *BaseDao[T]) SelectList(ctx context.Context, q *gplus.QueryCond[T], opts ...gplus.OptionFunc) ([]*T, error) {
	results, db := gplus.SelectList[T](ctx, q, opts...)
	return results, db.Error
}

// SelectPage 根据条件分页查询记录
func (b *BaseDao[T]) SelectPage(
	ctx context.Context, page *gplus.Page[T], q *gplus.QueryCond[T], opts ...gplus.OptionFunc) (
	*gplus.Page[T], error) {
	page, db := gplus.SelectPage[T](ctx, page, q, opts...)
	return page, db.Error
}

// SelectStreamingPage 根据条件分页查询记录
func (b *BaseDaoWithComparable[T, V]) SelectStreamingPage(
	ctx context.Context, page *gplus.StreamingPage[T, V], q *gplus.QueryCond[T], opts ...gplus.OptionFunc) (
	*gplus.StreamingPage[T, V], error) {
	result, db := gplus.SelectStreamingPage[T, V](ctx, page, q, opts...)
	if db.Error != nil {
		return nil, db.Error
	}
	return result, nil
}

// SelectCount 根据条件查询记录数量
func (b *BaseDao[T]) SelectCount(ctx context.Context, q *gplus.QueryCond[T], opts ...gplus.OptionFunc) (int64, error) {
	var count int64
	count, db := gplus.SelectCount[T](ctx, q, opts...)
	return count, db.Error
}

// Exists 根据条件判断记录是否存在
func (b *BaseDao[T]) Exists(ctx context.Context, q *gplus.QueryCond[T], opts ...gplus.OptionFunc) (bool, error) {
	count, err := b.SelectCount(ctx, q, opts...)
	return count > 0, err
}

// SelectPageGeneric 根据传入的泛型封装分页记录
// 第一个泛型代表数据库表实体
// 第二个泛型代表返回记录实体
func (b *BaseDaoGeneric[T, R]) SelectPageGeneric(
	ctx context.Context, page *gplus.Page[R], q *gplus.QueryCond[T], opts ...gplus.OptionFunc) (
	*gplus.Page[R], error) {
	result, db := gplus.SelectPageGeneric[T, R](ctx, page, q, opts...)
	if db.Error != nil {
		return nil, db.Error
	}
	return result, nil
}

// SelectStreamingPageGeneric 根据传入的泛型封装分页记录
// 第一个泛型代表数据库表实体
// 第二个泛型代表返回记录实体
func (b *BaseDaoStreamingGeneric[T, R, V]) SelectStreamingPageGeneric(
	ctx context.Context, page *gplus.StreamingPage[R, V], q *gplus.QueryCond[T], opts ...gplus.OptionFunc) (
	*gplus.StreamingPage[R, V], error) {
	result, db := gplus.SelectStreamingPageGeneric[T, R, V](ctx, page, q, opts...)
	if db.Error != nil {
		return nil, db.Error
	}
	return result, nil
}

// SelectGeneric 根据传入的泛型封装记录
// 第一个泛型代表数据库表实体
// 第二个泛型代表返回记录实体
func (b *BaseDaoGeneric[T, R]) SelectGeneric(ctx context.Context, q *gplus.QueryCond[T], opts ...gplus.OptionFunc) (
	R, error) {
	result, db := gplus.SelectGeneric[T, R](ctx, q, opts...)
	return result, db.Error
}

//---------------------------------------------------插入------------------------------------------------------//

// Insert 插入一条记录
func (b *BaseDao[T]) Insert(ctx context.Context, entity *T, opts ...gplus.OptionFunc) error {
	return gplus.Insert[T](ctx, entity, opts...).Error
}

// InsertBatch 批量插入多条记录
func (b *BaseDao[T]) InsertBatch(ctx context.Context, entities []*T, opts ...gplus.OptionFunc) error {
	return gplus.InsertBatch[T](ctx, entities, opts...).Error
}

// InsertBatchSize 批量插入多条记录
func (b *BaseDao[T]) InsertBatchSize(
	ctx context.Context, entities []*T, batchSize int, opts ...gplus.OptionFunc) error {
	return gplus.InsertBatchSize[T](ctx, entities, batchSize, opts...).Error
}

//---------------------------------------------------删除------------------------------------------------------//

// DeleteById 根据 ID 删除记录
func (b *BaseDao[T]) DeleteById(ctx context.Context, id any, opts ...gplus.OptionFunc) error {
	return gplus.DeleteById[T](ctx, id, opts...).Error
}

// DeleteByIds 根据 ID 批量删除记录
func (b *BaseDao[T]) DeleteByIds(ctx context.Context, ids any, opts ...gplus.OptionFunc) error {
	return gplus.DeleteByIds[T](ctx, ids, opts...).Error
}

// Delete 根据条件删除记录
func (b *BaseDao[T]) Delete(ctx context.Context, q *gplus.QueryCond[T], opts ...gplus.OptionFunc) error {
	return gplus.Delete[T](ctx, q, opts...).Error
}

//---------------------------------------------------更新------------------------------------------------------//

// UpdateById 根据 ID 更新,默认零值不更新
func (b *BaseDao[T]) UpdateById(ctx context.Context, entity *T, opts ...gplus.OptionFunc) error {
	opts = append(opts, gplus.Omit("create_time"))
	return gplus.UpdateById[T](ctx, entity, opts...).Error
}

// UpdateZeroById 根据 ID 零值更新
func (b *BaseDao[T]) UpdateZeroById(ctx context.Context, entity *T, opts ...gplus.OptionFunc) error {
	opts = append(opts, gplus.Omit("create_time"))
	return gplus.UpdateZeroById[T](ctx, entity, opts...).Error
}

// Update 根据 Map 更新
func (b *BaseDao[T]) Update(ctx context.Context, q *gplus.QueryCond[T], opts ...gplus.OptionFunc) error {
	opts = append(opts, gplus.Omit("create_time"))
	return gplus.Update[T](ctx, q, opts...).Error
}
