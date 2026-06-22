package repository

import (
    "cbt-api/internal/models"
    "gorm.io/gorm"
    "time"
)

type UserRepository struct {
    db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
    return &UserRepository{db: db}
}

func (r *UserRepository) FindByID(id string) (*models.User, error) {
    var user models.User
    err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&user).Error
    if err != nil {
        return nil, err
    }
    return &user, nil
}

func (r *UserRepository) FindByEmail(email string) (*models.User, error) {
    var user models.User
    err := r.db.Where("email = ? AND deleted_at IS NULL", email).First(&user).Error
    if err != nil {
        return nil, err
    }
    return &user, nil
}

func (r *UserRepository) Create(user *models.User) error {
    return r.db.Create(user).Error
}

func (r *UserRepository) Update(user *models.User) error {
    return r.db.Save(user).Error
}

func (r *UserRepository) Delete(id string) error {
    return r.db.Where("id = ?", id).Delete(&models.User{}).Error
}

func (r *UserRepository) FindByUsername(username string) (*models.User, error) {
    var user models.User
    err := r.db.Where("username = ? AND deleted_at IS NULL", username).First(&user).Error
    if err != nil {
        return nil, err
    }
    return &user, nil
}

func (r *UserRepository) CreateUser(user *models.User) error {
    return r.db.Create(user).Error
}

func (r *UserRepository) UpdateUser(user *models.User) error {
    return r.db.Save(user).Error
}

func (r *UserRepository) SoftDeleteUser(id string) error {
    return r.db.Model(&models.User{}).Where("id = ?", id).Update("deleted_at", time.Now()).Error
}

// package repository

// import (
//     "cbt-api/internal/models"
//     "gorm.io/gorm"
// )

// type UserRepository struct {
//     db *gorm.DB
// }

// func NewUserRepository(db *gorm.DB) *UserRepository {
//     return &UserRepository{db: db}
// }

// func (r *UserRepository) FindByID(id string) (*models.User, error) {
//     var user models.User
//     err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&user).Error
//     if err != nil {
//         return nil, err
//     }
//     return &user, nil
// }

// func (r *UserRepository) FindByEmail(email string) (*models.User, error) {
//     var user models.User
//     err := r.db.Where("email = ? AND deleted_at IS NULL", email).First(&user).Error
//     if err != nil {
//         return nil, err
//     }
//     return &user, nil
// }

// func (r *UserRepository) Create(user *models.User) error {
//     return r.db.Create(user).Error
// }

// func (r *UserRepository) Update(user *models.User) error {
//     return r.db.Save(user).Error
// }

// func (r *UserRepository) Delete(id string) error {
//     return r.db.Where("id = ?", id).Delete(&models.User{}).Error
// }


