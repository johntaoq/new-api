package model

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"

	"github.com/bytedance/gopkg/util/gopool"
	"gorm.io/gorm"
)

const UserNameMaxLength = 20

// User if you add sensitive fields, don't forget to clean them in setupLogin function.
// Otherwise, the sensitive information will be saved on local storage in plain text!
type User struct {
	Id               int            `json:"id"`
	Username         string         `json:"username" gorm:"unique;index" validate:"max=20"`
	Password         string         `json:"password" gorm:"not null;" validate:"min=8,max=20"`
	OriginalPassword string         `json:"original_password" gorm:"-:all"` // this field is only for Password change verification, don't save it to database!
	DisplayName      string         `json:"display_name" gorm:"index" validate:"max=20"`
	Role             int            `json:"role" gorm:"type:int;default:1"` // admin, common
	StaffRole        string         `json:"staff_role" gorm:"type:varchar(32);default:'';column:staff_role"`
	Status           int            `json:"status" gorm:"type:int;default:1"` // enabled, disabled
	Email            string         `json:"email" gorm:"index" validate:"max=50"`
	GitHubId         string         `json:"github_id" gorm:"column:github_id;index"`
	DiscordId        string         `json:"discord_id" gorm:"column:discord_id;index"`
	OidcId           string         `json:"oidc_id" gorm:"column:oidc_id;index"`
	WeChatId         string         `json:"wechat_id" gorm:"column:wechat_id;index"`
	TelegramId       string         `json:"telegram_id" gorm:"column:telegram_id;index"`
	VerificationCode string         `json:"verification_code" gorm:"-:all"`                                    // this field is only for Email verification, don't save it to database!
	AccessToken      *string        `json:"access_token" gorm:"type:char(32);column:access_token;uniqueIndex"` // this token is for system management
	Quota            int            `json:"quota" gorm:"type:int;default:0"`
	PaidQuota        int            `json:"paid_quota" gorm:"type:int;default:0;column:paid_quota"`
	GiftQuota        int            `json:"gift_quota" gorm:"type:int;default:0;column:gift_quota"`
	UsedQuota        int            `json:"used_quota" gorm:"type:int;default:0;column:used_quota"` // used quota
	RequestCount     int            `json:"request_count" gorm:"type:int;default:0;"`               // request number
	Group            string         `json:"group" gorm:"type:varchar(64);default:'default'"`
	AffCode          string         `json:"aff_code" gorm:"type:varchar(32);column:aff_code;uniqueIndex"`
	AffCount         int            `json:"aff_count" gorm:"type:int;default:0;column:aff_count"`
	AffQuota         int            `json:"aff_quota" gorm:"type:int;default:0;column:aff_quota"`
	AffHistoryQuota  int            `json:"aff_history_quota" gorm:"type:int;default:0;column:aff_history"`
	InviterId        int            `json:"inviter_id" gorm:"type:int;column:inviter_id;index"`
	DeletedAt        gorm.DeletedAt `gorm:"index"`
	LinuxDOId        string         `json:"linux_do_id" gorm:"column:linux_do_id;index"`
	Setting          string         `json:"setting" gorm:"type:text;column:setting"`
	Remark           string         `json:"remark,omitempty" gorm:"type:varchar(255)" validate:"max=255"`
	StripeCustomer   string         `json:"stripe_customer" gorm:"type:varchar(64);column:stripe_customer;index"`
}

func (user *User) ToBaseUser() *UserBase {
	cache := &UserBase{
		Id:        user.Id,
		Group:     user.Group,
		Quota:     user.Quota,
		PaidQuota: user.PaidQuota,
		GiftQuota: user.GiftQuota,
		Status:    user.Status,
		Username:  user.Username,
		StaffRole: user.StaffRole,
		Setting:   user.Setting,
		Email:     user.Email,
	}
	return cache
}

func (user *User) GetAccessToken() string {
	if user.AccessToken == nil {
		return ""
	}
	return *user.AccessToken
}

func (user *User) SetAccessToken(token string) {
	user.AccessToken = &token
}

func (user *User) GetSetting() dto.UserSetting {
	setting := dto.UserSetting{}
	if user.Setting != "" {
		err := common.Unmarshal([]byte(user.Setting), &setting)
		if err != nil {
			common.SysLog("failed to unmarshal setting: " + err.Error())
		}
	}
	return setting
}

func (user *User) SetSetting(setting dto.UserSetting) {
	settingBytes, err := common.Marshal(setting)
	if err != nil {
		common.SysLog("failed to marshal setting: " + err.Error())
		return
	}
	user.Setting = string(settingBytes)
}

func generateDefaultSidebarConfigForRole(userRole int, staffRole string) string {
	defaultConfig := BuildSidebarPermissionModules(userRole, staffRole)
	configBytes, err := common.Marshal(defaultConfig)
	if err != nil {
		common.SysLog("failed to marshal default sidebar config: " + err.Error())
		return ""
	}
	return string(configBytes)
}

// CheckUserExistOrDeleted check if user exist or deleted, if not exist, return false, nil, if deleted or exist, return true, nil
func CheckUserExistOrDeleted(username string, email string) (bool, error) {
	var user User

	// err := DB.Unscoped().First(&user, "username = ? or email = ?", username, email).Error
	// check email if empty
	var err error
	if email == "" {
		err = DB.Unscoped().First(&user, "username = ?", username).Error
	} else {
		err = DB.Unscoped().First(&user, "username = ? or email = ?", username, email).Error
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// not exist, return false, nil
			return false, nil
		}
		// other error, return false, err
		return false, err
	}
	// exist, return true, nil
	return true, nil
}

func GetMaxUserId() int {
	var user User
	DB.Unscoped().Last(&user)
	return user.Id
}

func GetAllUsers(pageInfo *common.PageInfo) (users []*User, total int64, err error) {
	// Start transaction
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Get total count within transaction
	err = tx.Unscoped().Model(&User{}).Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// Get paginated users within same transaction
	err = tx.Unscoped().Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Omit("password").Find(&users).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// Commit transaction
	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func SearchUsers(keyword string, group string, startIdx int, num int) ([]*User, int64, error) {
	var users []*User
	var total int64
	var err error

	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	query := tx.Unscoped().Model(&User{})
	likeCondition := "username LIKE ? OR email LIKE ? OR display_name LIKE ?"

	keywordInt, atoiErr := strconv.Atoi(keyword)
	if atoiErr == nil {
		likeCondition = "id = ? OR " + likeCondition
		if group != "" {
			query = query.Where("("+likeCondition+") AND "+commonGroupCol+" = ?", keywordInt, "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", group)
		} else {
			query = query.Where(likeCondition, keywordInt, "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
		}
	} else {
		if group != "" {
			query = query.Where("("+likeCondition+") AND "+commonGroupCol+" = ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", group)
		} else {
			query = query.Where(likeCondition, "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
		}
	}

	err = query.Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	err = query.Omit("password").Order("id desc").Limit(num).Offset(startIdx).Find(&users).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func GetUserById(id int, selectAll bool) (*User, error) {
	if id == 0 {
		return nil, errors.New("id is empty")
	}
	user := User{Id: id}
	var err error = nil
	if selectAll {
		err = DB.First(&user, "id = ?", id).Error
	} else {
		err = DB.Omit("password").First(&user, "id = ?", id).Error
	}
	return &user, err
}

func GetUserIdByAffCode(affCode string) (int, error) {
	if affCode == "" {
		return 0, errors.New("affCode is empty")
	}
	var user User
	err := DB.Select("id").First(&user, "aff_code = ?", affCode).Error
	return user.Id, err
}

func DeleteUserById(id int) (err error) {
	if id == 0 {
		return errors.New("id is empty")
	}
	user := User{Id: id}
	return user.Delete()
}

func HardDeleteUserById(id int) error {
	if id == 0 {
		return errors.New("id is empty")
	}
	err := DB.Unscoped().Delete(&User{}, "id = ?", id).Error
	return err
}

func inviteUser(inviterId int) (err error) {
	user, err := GetUserById(inviterId, true)
	if err != nil {
		return err
	}
	user.AffCount++
	user.AffQuota += common.QuotaForInviter
	user.AffHistoryQuota += common.QuotaForInviter
	return DB.Save(user).Error
}

func (user *User) legacyTransferAffQuotaToQuota(quota int) error {
	if float64(quota) < common.QuotaPerUnit {
		return fmt.Errorf("transfer quota minimum is %s", logger.LogQuota(int(common.QuotaPerUnit)))
	}

	tx := DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer tx.Rollback()

	if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&user, user.Id).Error; err != nil {
		return err
	}
	if user.AffQuota < quota {
		return errors.New("affiliate quota is not enough")
	}

	user.AffQuota -= quota
	if err := tx.Save(user).Error; err != nil {
		return err
	}

	now := common.GetTimestamp()
	if err := tx.Create(&UserQuotaFunding{
		UserId:                    user.Id,
		FundingType:               QuotaFundingTypeGift,
		SourceType:                QuotaFundingSourceInviteReward,
		SourceName:                "affiliate transfer",
		GrantedQuota:              quota,
		RemainingQuota:            quota,
		RecognizedRevenueUSDTotal: 0,
		QuotaPerUnitSnapshot:      common.QuotaPerUnit,
		CreatedAt:                 now,
		UpdatedAt:                 now,
	}).Error; err != nil {
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}
	return updateUserCache(*user)
}

func (user *User) TransferAffQuotaToQuota(quota int) error {
	if float64(quota) < common.QuotaPerUnit {
		return fmt.Errorf("transfer quota minimum is %s", logger.LogQuota(int(common.QuotaPerUnit)))
	}

	tx := DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer tx.Rollback()

	if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&user, user.Id).Error; err != nil {
		return err
	}
	if user.AffQuota < quota {
		return errors.New("affiliate quota is not enough")
	}

	user.AffQuota -= quota
	if err := tx.Save(user).Error; err != nil {
		return err
	}

	now := common.GetTimestamp()
	updatedUser, err := grantUserQuotaTx(tx, QuotaFundingGrantParams{
		UserId:           user.Id,
		FundingType:      QuotaFundingTypeGift,
		SourceType:       QuotaFundingSourceInviteReward,
		SourceName:       "affiliate transfer",
		GrantedQuota:     quota,
		Remark:           "affiliate transfer to wallet",
		BalanceEntryType: CustomerMonthlyStatementEntryTypeGift,
		OccurredAt:       now,
	})
	if err != nil {
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}
	*user = updatedUser
	return updateUserCache(updatedUser)
}

func (user *User) Insert(inviterId int) error {
	var err error
	if user.Password != "" {
		user.Password, err = common.Password2Hash(user.Password)
		if err != nil {
			return err
		}
	}
	user.Quota = 0
	user.PaidQuota = 0
	user.GiftQuota = 0
	user.AffCode = common.GetRandomString(4)

	if user.Setting == "" {
		defaultSetting := dto.UserSetting{}
		user.SetSetting(defaultSetting)
	}

	result := DB.Create(user)
	if result.Error != nil {
		return result.Error
	}

	var createdUser User
	if err := DB.Where("username = ?", user.Username).First(&createdUser).Error; err == nil {
		defaultSidebarConfig := generateDefaultSidebarConfigForRole(createdUser.Role, createdUser.StaffRole)
		if defaultSidebarConfig != "" {
			currentSetting := createdUser.GetSetting()
			currentSetting.SidebarModules = defaultSidebarConfig
			createdUser.SetSetting(currentSetting)
			createdUser.Update(false)
			common.SysLog(fmt.Sprintf("initialized sidebar config for user %s (role=%d staff_role=%s)", createdUser.Username, createdUser.Role, createdUser.StaffRole))
		}
	}

	if common.QuotaForNewUser > 0 {
		_ = GrantGiftQuota(user.Id, common.QuotaForNewUser, QuotaFundingSourceSignupBonus, 0, "", "new user signup bonus")
		RecordLog(user.Id, LogTypeSystem, fmt.Sprintf("signup bonus granted: %s", logger.LogQuota(common.QuotaForNewUser)))
	}
	if inviterId != 0 {
		if common.QuotaForInvitee > 0 {
			_ = GrantGiftQuota(user.Id, common.QuotaForInvitee, QuotaFundingSourceInviteReward, inviterId, "", "invitee bonus")
			RecordLog(user.Id, LogTypeSystem, fmt.Sprintf("invite bonus granted: %s", logger.LogQuota(common.QuotaForInvitee)))
		}
		if common.QuotaForInviter > 0 {
			RecordLog(inviterId, LogTypeSystem, fmt.Sprintf("inviter reward granted: %s", logger.LogQuota(common.QuotaForInviter)))
			_ = inviteUser(inviterId)
		}
	}
	return nil
}

// InsertWithTx inserts a new user within an existing transaction.
// This is used for OAuth registration where user creation and binding need to be atomic.
// Post-creation tasks (sidebar config, logs, inviter rewards) are handled after the transaction commits.
func (user *User) InsertWithTx(tx *gorm.DB, inviterId int) error {
	var err error
	if user.Password != "" {
		user.Password, err = common.Password2Hash(user.Password)
		if err != nil {
			return err
		}
	}
	user.Quota = 0
	user.PaidQuota = 0
	user.GiftQuota = 0
	user.AffCode = common.GetRandomString(4)

	if user.Setting == "" {
		defaultSetting := dto.UserSetting{}
		user.SetSetting(defaultSetting)
	}

	result := tx.Create(user)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// FinalizeOAuthUserCreation performs post-transaction tasks for OAuth user creation.
// This should be called after the transaction commits successfully.
func (user *User) FinalizeOAuthUserCreation(inviterId int) {
	var createdUser User
	if err := DB.Where("id = ?", user.Id).First(&createdUser).Error; err == nil {
		defaultSidebarConfig := generateDefaultSidebarConfigForRole(createdUser.Role, createdUser.StaffRole)
		if defaultSidebarConfig != "" {
			currentSetting := createdUser.GetSetting()
			currentSetting.SidebarModules = defaultSidebarConfig
			createdUser.SetSetting(currentSetting)
			createdUser.Update(false)
			common.SysLog(fmt.Sprintf("initialized sidebar config for oauth user %s (role=%d staff_role=%s)", createdUser.Username, createdUser.Role, createdUser.StaffRole))
		}
	}

	if common.QuotaForNewUser > 0 {
		_ = GrantGiftQuota(user.Id, common.QuotaForNewUser, QuotaFundingSourceSignupBonus, 0, "", "oauth signup bonus")
		RecordLog(user.Id, LogTypeSystem, fmt.Sprintf("oauth signup bonus granted: %s", logger.LogQuota(common.QuotaForNewUser)))
	}
	if inviterId != 0 {
		if common.QuotaForInvitee > 0 {
			_ = GrantGiftQuota(user.Id, common.QuotaForInvitee, QuotaFundingSourceInviteReward, inviterId, "", "invitee bonus")
			RecordLog(user.Id, LogTypeSystem, fmt.Sprintf("oauth invite bonus granted: %s", logger.LogQuota(common.QuotaForInvitee)))
		}
		if common.QuotaForInviter > 0 {
			RecordLog(inviterId, LogTypeSystem, fmt.Sprintf("oauth inviter reward granted: %s", logger.LogQuota(common.QuotaForInviter)))
			_ = inviteUser(inviterId)
		}
	}
}

func (user *User) Update(updatePassword bool) error {
	var err error
	if updatePassword {
		user.Password, err = common.Password2Hash(user.Password)
		if err != nil {
			return err
		}
	}
	newUser := *user
	var origin User
	if err = DB.First(&origin, user.Id).Error; err != nil {
		return err
	}
	newUser.Quota = origin.Quota
	newUser.PaidQuota = origin.PaidQuota
	newUser.GiftQuota = origin.GiftQuota
	if err = DB.Model(&origin).Updates(newUser).Error; err != nil {
		return err
	}
	if user.Quota != origin.Quota {
		delta := user.Quota - origin.Quota
		if delta > 0 {
			if err = GrantGiftQuota(origin.Id, delta, QuotaFundingSourceAdminGrant, 0, "", "direct user update"); err != nil {
				return err
			}
		} else if delta < 0 {
			if _, err = ConsumeUserQuotaWithAllocation(origin.Id, -delta); err != nil {
				return err
			}
		}
	}
	updatedUser, err := GetUserById(user.Id, true)
	if err != nil {
		return err
	}
	*user = *updatedUser

	// Update cache
	return updateUserCache(*updatedUser)
}

func (user *User) Edit(updatePassword bool) error {
	var err error
	if updatePassword {
		user.Password, err = common.Password2Hash(user.Password)
		if err != nil {
			return err
		}
	}

	newUser := *user
	var origin User
	if err = DB.First(&origin, user.Id).Error; err != nil {
		return err
	}
	updates := map[string]interface{}{
		"username":     newUser.Username,
		"display_name": newUser.DisplayName,
		"group":        newUser.Group,
		"remark":       newUser.Remark,
		"role":         newUser.Role,
		"staff_role":   common.NormalizeStaffRole(newUser.StaffRole),
	}
	if updatePassword {
		updates["password"] = newUser.Password
	}

	if err = DB.Model(&origin).Updates(updates).Error; err != nil {
		return err
	}
	if newUser.Quota != origin.Quota {
		delta := newUser.Quota - origin.Quota
		if delta > 0 {
			if err = GrantGiftQuota(origin.Id, delta, QuotaFundingSourceAdminGrant, 0, "", "admin adjusted user quota"); err != nil {
				return err
			}
		} else if delta < 0 {
			if _, err = ConsumeUserQuotaWithAllocation(origin.Id, -delta); err != nil {
				return err
			}
		}
	}
	updatedUser, err := GetUserById(user.Id, true)
	if err != nil {
		return err
	}
	*user = *updatedUser

	// Update cache
	return updateUserCache(*updatedUser)
}

func (user *User) ClearBinding(bindingType string) error {
	if user.Id == 0 {
		return errors.New("user id is empty")
	}

	bindingColumnMap := map[string]string{
		"email":    "email",
		"github":   "github_id",
		"discord":  "discord_id",
		"oidc":     "oidc_id",
		"wechat":   "wechat_id",
		"telegram": "telegram_id",
		"linuxdo":  "linux_do_id",
	}

	column, ok := bindingColumnMap[bindingType]
	if !ok {
		return errors.New("invalid binding type")
	}

	if err := DB.Model(&User{}).Where("id = ?", user.Id).Update(column, "").Error; err != nil {
		return err
	}

	if err := DB.Where("id = ?", user.Id).First(user).Error; err != nil {
		return err
	}

	return updateUserCache(*user)
}

func (user *User) Delete() error {
	if user.Id == 0 {
		return errors.New("user id is empty")
	}
	if err := DB.Delete(user).Error; err != nil {
		return err
	}

	// Invalidate cached user data after soft delete.
	return invalidateUserCache(user.Id)
}

func (user *User) HardDelete() error {
	if user.Id == 0 {
		return errors.New("user id is empty")
	}
	err := DB.Unscoped().Delete(user).Error
	return err
}

// ValidateAndFill check password & user status
func (user *User) ValidateAndFill() (err error) {
	// When querying with struct, GORM will only query with non-zero fields,
	// that means if your field's value is 0, '', false or other zero values,
	// it won't be used to build query conditions
	password := user.Password
	username := strings.TrimSpace(user.Username)
	if username == "" || password == "" {
		return errors.New("用户名或密码为空")
	}
	// find buy username or email
	DB.Where("username = ? OR email = ?", username, username).First(user)
	okay := common.ValidatePasswordAndHash(password, user.Password)
	if !okay || user.Status != common.UserStatusEnabled {
		return errors.New("用户名或密码错误，或用户已被封禁")
	}
	return nil
}

func (user *User) FillUserById() error {
	if user.Id == 0 {
		return errors.New("id is empty")
	}
	DB.Where(User{Id: user.Id}).First(user)
	return nil
}

func (user *User) FillUserByEmail() error {
	if user.Email == "" {
		return errors.New("email is empty")
	}
	DB.Where(User{Email: user.Email}).First(user)
	return nil
}

func (user *User) FillUserByGitHubId() error {
	if user.GitHubId == "" {
		return errors.New("GitHub id is empty")
	}
	DB.Where(User{GitHubId: user.GitHubId}).First(user)
	return nil
}

// UpdateGitHubId updates the user's GitHub ID (used for migration from login to numeric ID)
func (user *User) UpdateGitHubId(newGitHubId string) error {
	if user.Id == 0 {
		return errors.New("user id is empty")
	}
	return DB.Model(user).Update("github_id", newGitHubId).Error
}

func (user *User) FillUserByDiscordId() error {
	if user.DiscordId == "" {
		return errors.New("discord id is empty")
	}
	DB.Where(User{DiscordId: user.DiscordId}).First(user)
	return nil
}

func (user *User) FillUserByOidcId() error {
	if user.OidcId == "" {
		return errors.New("oidc id is empty")
	}
	DB.Where(User{OidcId: user.OidcId}).First(user)
	return nil
}

func (user *User) FillUserByWeChatId() error {
	if user.WeChatId == "" {
		return errors.New("WeChat id is empty")
	}
	DB.Where(User{WeChatId: user.WeChatId}).First(user)
	return nil
}

func (user *User) FillUserByTelegramId() error {
	if user.TelegramId == "" {
		return errors.New("Telegram id is empty")
	}
	err := DB.Where(User{TelegramId: user.TelegramId}).First(user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.New("Telegram account is not bound")
	}
	return nil
}

func IsEmailAlreadyTaken(email string) bool {
	return DB.Unscoped().Where("email = ?", email).Find(&User{}).RowsAffected == 1
}

func IsWeChatIdAlreadyTaken(wechatId string) bool {
	return DB.Unscoped().Where("wechat_id = ?", wechatId).Find(&User{}).RowsAffected == 1
}

func IsGitHubIdAlreadyTaken(githubId string) bool {
	return DB.Unscoped().Where("github_id = ?", githubId).Find(&User{}).RowsAffected == 1
}

func IsDiscordIdAlreadyTaken(discordId string) bool {
	return DB.Unscoped().Where("discord_id = ?", discordId).Find(&User{}).RowsAffected == 1
}

func IsOidcIdAlreadyTaken(oidcId string) bool {
	return DB.Where("oidc_id = ?", oidcId).Find(&User{}).RowsAffected == 1
}

func IsTelegramIdAlreadyTaken(telegramId string) bool {
	return DB.Unscoped().Where("telegram_id = ?", telegramId).Find(&User{}).RowsAffected == 1
}

func ResetUserPasswordByEmail(email string, password string) error {
	if email == "" || password == "" {
		return errors.New("email or password is empty")
	}
	hashedPassword, err := common.Password2Hash(password)
	if err != nil {
		return err
	}
	err = DB.Model(&User{}).Where("email = ?", email).Update("password", hashedPassword).Error
	return err
}

func IsAdmin(userId int) bool {
	if userId == 0 {
		return false
	}
	var user User
	err := DB.Where("id = ?", userId).Select("role").Find(&user).Error
	if err != nil {
		common.SysLog("no such user " + err.Error())
		return false
	}
	return user.Role >= common.RoleAdminUser
}

//// IsUserEnabled checks user status from Redis first, falls back to DB if needed
//func IsUserEnabled(id int, fromDB bool) (status bool, err error) {
//	defer func() {
//		// Update Redis cache asynchronously on successful DB read
//		if shouldUpdateRedis(fromDB, err) {
//			gopool.Go(func() {
//				if err := updateUserStatusCache(id, status); err != nil {
//					common.SysError("failed to update user status cache: " + err.Error())
//				}
//			})
//		}
//	}()
//	if !fromDB && common.RedisEnabled {
//		// Try Redis first
//		status, err := getUserStatusCache(id)
//		if err == nil {
//			return status == common.UserStatusEnabled, nil
//		}
//		// Don't return error - fall through to DB
//	}
//	fromDB = true
//	var user User
//	err = DB.Where("id = ?", id).Select("status").Find(&user).Error
//	if err != nil {
//		return false, err
//	}
//
//	return user.Status == common.UserStatusEnabled, nil
//}

func ValidateAccessToken(token string) (user *User) {
	if token == "" {
		return nil
	}
	token = strings.Replace(token, "Bearer ", "", 1)
	user = &User{}
	if DB.Where("access_token = ?", token).First(user).RowsAffected == 1 {
		return user
	}
	return nil
}

// GetUserQuota gets quota from Redis first, falls back to DB if needed
func GetUserQuota(id int, fromDB bool) (quota int, err error) {
	defer func() {
		// Update Redis cache asynchronously on successful DB read
		if shouldUpdateRedis(fromDB, err) {
			gopool.Go(func() {
				if err := updateUserQuotaCache(id, quota); err != nil {
					common.SysLog("failed to update user quota cache: " + err.Error())
				}
			})
		}
	}()
	if !fromDB && common.RedisEnabled {
		quota, err := getUserQuotaCache(id)
		if err == nil {
			return quota, nil
		}
		// Don't return error - fall through to DB
	}
	fromDB = true
	err = DB.Model(&User{}).Where("id = ?", id).Select("quota").Find(&quota).Error
	if err != nil {
		return 0, err
	}

	return quota, nil
}

func GetUserUsedQuota(id int) (quota int, err error) {
	err = DB.Model(&User{}).Where("id = ?", id).Select("used_quota").Find(&quota).Error
	return quota, err
}

func GetUserEmail(id int) (email string, err error) {
	err = DB.Model(&User{}).Where("id = ?", id).Select("email").Find(&email).Error
	return email, err
}

// GetUserGroup gets group from Redis first, falls back to DB if needed
func GetUserGroup(id int, fromDB bool) (group string, err error) {
	defer func() {
		// Update Redis cache asynchronously on successful DB read
		if shouldUpdateRedis(fromDB, err) {
			gopool.Go(func() {
				if err := updateUserGroupCache(id, group); err != nil {
					common.SysLog("failed to update user group cache: " + err.Error())
				}
			})
		}
	}()
	if !fromDB && common.RedisEnabled {
		group, err := getUserGroupCache(id)
		if err == nil {
			return group, nil
		}
		// Don't return error - fall through to DB
	}
	fromDB = true
	err = DB.Model(&User{}).Where("id = ?", id).Select(commonGroupCol).Find(&group).Error
	if err != nil {
		return "", err
	}

	return group, nil
}

// GetUserSetting gets setting from Redis first, falls back to DB if needed
func GetUserSetting(id int, fromDB bool) (settingMap dto.UserSetting, err error) {
	var setting string
	defer func() {
		// Update Redis cache asynchronously on successful DB read
		if shouldUpdateRedis(fromDB, err) {
			gopool.Go(func() {
				if err := updateUserSettingCache(id, setting); err != nil {
					common.SysLog("failed to update user setting cache: " + err.Error())
				}
			})
		}
	}()
	if !fromDB && common.RedisEnabled {
		setting, err := getUserSettingCache(id)
		if err == nil {
			return setting, nil
		}
		// Don't return error - fall through to DB
	}
	fromDB = true
	// can be nil setting
	var safeSetting sql.NullString
	err = DB.Model(&User{}).Where("id = ?", id).Select("setting").Find(&safeSetting).Error
	if err != nil {
		return settingMap, err
	}
	if safeSetting.Valid {
		setting = safeSetting.String
	} else {
		setting = ""
	}
	userBase := &UserBase{
		Setting: setting,
	}
	return userBase.GetSetting(), nil
}

func IncreaseUserQuota(id int, quota int, db bool) (err error) {
	if quota < 0 {
		return errors.New("quota must be non-negative")
	}
	return GrantQuotaFunding(QuotaFundingGrantParams{
		UserId:            id,
		FundingType:       QuotaFundingTypePaid,
		SourceType:        QuotaFundingSourceSystemAdjust,
		GrantedQuota:      quota,
		RevenueUSD:        quotaToUSDWithSnapshot(quota, common.QuotaPerUnit),
		Remark:            "legacy increase",
		SkipBalanceLedger: true,
	})
}

func increaseUserQuota(id int, quota int) (err error) {
	return IncreaseUserQuota(id, quota, true)
}

func DecreaseUserQuota(id int, quota int) (err error) {
	if quota < 0 {
		return errors.New("quota must be non-negative")
	}
	_, err = ConsumeUserQuotaWithAllocation(id, quota)
	return err
}

func decreaseUserQuota(id int, quota int) (err error) {
	return DecreaseUserQuota(id, quota)
}

func DeltaUpdateUserQuota(id int, delta int) (err error) {
	if delta == 0 {
		return nil
	}
	if delta > 0 {
		return IncreaseUserQuota(id, delta, false)
	} else {
		return DecreaseUserQuota(id, -delta)
	}
}

//func GetRootUserEmail() (email string) {
//	DB.Model(&User{}).Where("role = ?", common.RoleRootUser).Select("email").Find(&email)
//	return email
//}

func GetRootUser() (user *User) {
	DB.Where("role = ?", common.RoleRootUser).First(&user)
	return user
}

func UpdateUserUsedQuotaAndRequestCount(id int, quota int) {
	if common.BatchUpdateEnabled {
		addNewRecord(BatchUpdateTypeUsedQuota, id, quota)
		addNewRecord(BatchUpdateTypeRequestCount, id, 1)
		return
	}
	updateUserUsedQuotaAndRequestCount(id, quota, 1)
}

func updateUserUsedQuotaAndRequestCount(id int, quota int, count int) {
	err := DB.Model(&User{}).Where("id = ?", id).Updates(
		map[string]interface{}{
			"used_quota":    gorm.Expr("used_quota + ?", quota),
			"request_count": gorm.Expr("request_count + ?", count),
		},
	).Error
	if err != nil {
		common.SysLog("failed to update user used quota and request count: " + err.Error())
		return
	}

	//// 闂傚倷绀侀幖顐⒚洪妶澶嬪仱闁靛ň鏅涢拑鐔封攽閻樻彃顏痪鍙ョ矙閺岀喖骞嗚閿涘秹鏌?	//if err := invalidateUserCache(id); err != nil {
	//	common.SysError("failed to invalidate user cache: " + err.Error())
	//}
}

func updateUserUsedQuota(id int, quota int) {
	err := DB.Model(&User{}).Where("id = ?", id).Updates(
		map[string]interface{}{
			"used_quota": gorm.Expr("used_quota + ?", quota),
		},
	).Error
	if err != nil {
		common.SysLog("failed to update user used quota: " + err.Error())
	}
}

func updateUserRequestCount(id int, count int) {
	err := DB.Model(&User{}).Where("id = ?", id).Update("request_count", gorm.Expr("request_count + ?", count)).Error
	if err != nil {
		common.SysLog("failed to update user request count: " + err.Error())
	}
}

// GetUsernameById gets username from Redis first, falls back to DB if needed
func GetUsernameById(id int, fromDB bool) (username string, err error) {
	defer func() {
		// Update Redis cache asynchronously on successful DB read
		if shouldUpdateRedis(fromDB, err) {
			gopool.Go(func() {
				if err := updateUserNameCache(id, username); err != nil {
					common.SysLog("failed to update user name cache: " + err.Error())
				}
			})
		}
	}()
	if !fromDB && common.RedisEnabled {
		username, err := getUserNameCache(id)
		if err == nil {
			return username, nil
		}
		// Don't return error - fall through to DB
	}
	fromDB = true
	err = DB.Model(&User{}).Where("id = ?", id).Select("username").Find(&username).Error
	if err != nil {
		return "", err
	}

	return username, nil
}

func IsLinuxDOIdAlreadyTaken(linuxDOId string) bool {
	var user User
	err := DB.Unscoped().Where("linux_do_id = ?", linuxDOId).First(&user).Error
	return !errors.Is(err, gorm.ErrRecordNotFound)
}

func (user *User) FillUserByLinuxDOId() error {
	if user.LinuxDOId == "" {
		return errors.New("linux do id is empty")
	}
	err := DB.Where("linux_do_id = ?", user.LinuxDOId).First(user).Error
	return err
}

func RootUserExists() bool {
	var user User
	err := DB.Where("role = ?", common.RoleRootUser).First(&user).Error
	if err != nil {
		return false
	}
	return true
}
