// internal/infrastructure/persistence/redis_storage/errors.go(переименован)
package redis_storage

// Ошибки хранилища
var (
	ErrSymbolNotFound  = StorageError{"символ не найден"}
	ErrStorageFull     = StorageError{"хранилище переполнено"}
	ErrInvalidLimit    = StorageError{"неверный лимит"}
	ErrAlreadyExists   = StorageError{"символ уже существует"}
	ErrSubscriberError = StorageError{"ошибка подписчика"}
	ErrRedisNotReady   = StorageError{"Redis не готов"}
)

// StorageError ошибка хранилища
type StorageError struct {
	Message string
}

func (e StorageError) Error() string {
	return e.Message
}
