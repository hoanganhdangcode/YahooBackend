package main

import (
	"UserService/config"
	"UserService/handler"
	"UserService/model"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

//==========================================================

func main() {
	app := fiber.New()
	config.ConnectDB()
	config.ConnectRedis()
	config.ConnectFirebase()
	config.Taobanguser()
	config.Taobangtoken()
	config.Taobangpost()
	config.Taobangchitietpost()
	var otpQueue = make(chan model.OTP, 100)
	// for i := 0; i < 5; i++ {
	// 	otp := model.OTP{Phone: "0396320369", Otp: fmt.Sprintf("123%d", i)}
	// 	// otpQueue <- otp
	// 	fmt.Println("Pushed:", otp)
	// 	time.Sleep(time.Second)
	// }
	app.Post("/sendotp", func(c *fiber.Ctx) error {
		type OtpInput struct {
			Phonenumber string `json:"phone"`
		}
		var input OtpInput
		if err := c.BodyParser(&input); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid input",
			})
		}

		// Kiểm tra số điện thoại tồn tại
		var user model.User
		err := config.DB.Where("phone = ?", input.Phonenumber).First(&user).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Phone number not found",
			})
		}
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Database error: " + err.Error(),
			})
		}

		// Sinh OTP và mã hóa
		otp := genOTP() // ví dụ trả về "123456"
		hash, err := bcrypt.GenerateFromPassword([]byte(otp), bcrypt.DefaultCost)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to generate OTP",
			})
		}

		redisKey := "otp:" + strconv.FormatUint(uint64(user.Id), 10)
		// Lưu hash vào Redis với TTL
		err = config.Redis.Set(c.Context(), redisKey, hash, 1*time.Minute).Err()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Redis error: " + err.Error(),
			})
		}

		// Gửi otp sang hàng đợi nếu cần
		otpQueue <- model.OTP{Phone: input.Phonenumber, Otp: otp}
		log.Println("Đã tạo OTP:", otp, "với ID:", user.Id)

		// Trả lại ID cho client (OTP không nên gửi nếu là production)
		return c.JSON(fiber.Map{
			"message": "OTP sent successfully",
			"id":      user.Id,
		})
	})
	app.Post("/verifyotp", func(c *fiber.Ctx) error {
		log.Println("Đang verify otp")
		type VerifyInput struct {
			Id        uint   `json:"id"`
			Otpsubmit string `json:"otpsubmit"`
		}
		var input VerifyInput
		if err := c.BodyParser(&input); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid input",
			})
		}

		redisKey := "otp:" + strconv.FormatUint(uint64(input.Id), 10)
		hash, err := config.Redis.Get(c.Context(), redisKey).Result()
		if err == redis.Nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "OTP not found or expired",
			})
		} else if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Redis error: " + err.Error(),
			})
		}
		err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(strings.TrimSpace(input.Otpsubmit)))
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid OTP",
			})
		}
		log.Println("OTP verified successfully for otpid:" + strconv.FormatUint(uint64(input.Id), 10))

		tokenchangepass, err := handler.GenerateRefreshToken()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to generate change password token",
			})
		}
		rediskeychangepass := "changepass:" + tokenchangepass
		// Lưu hash vào Redis với TTL
		err = config.Redis.Set(c.Context(), rediskeychangepass, strconv.FormatUint(uint64(input.Id), 10), 5*time.Minute).Err()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Redis error: " + err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"message": "OTP verified successfully",
			"token":   tokenchangepass,
		})
	})

	app.Post("/updatepassword", func(c *fiber.Ctx) error {
		type UpdatePassInput struct {
			Id              uint   `json:"id"`
			NewPassword     string `json:"newpass"`
			Tokenchangepass string `json:"token"`
		}

		var input UpdatePassInput
		if err := c.BodyParser(&input); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid input",
			})
		}
		rediskey := "changepass:" + input.Tokenchangepass
		// Kiểm tra token trong Redis
		id, err := config.Redis.Get(c.Context(), rediskey).Result()
		if err == redis.Nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Change password token not found or expired",
			})
		} else if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Redis error: " + err.Error(),
			})
		}
		if strconv.FormatUint(uint64(input.Id), 10) != id {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid change password token",
			})
		}
		// Kiểm tra xem ID có tồn tại trong DB không
		var user model.User
		if err := config.DB.First(&user, input.Id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": "User not found",
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Database error: " + err.Error(),
			})
		}
		// Kiểm tra mật khẩu mới không được để trống
		if input.NewPassword == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Mật khẩu không được để trống",
			})
		}

		// Mã hóa mật khẩu mới
		hashed, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Hash error: " + err.Error(),
			})
		}
		// Cập nhật DB theo ID
		if err := config.DB.Model(&model.User{}).
			Where("id = ?", input.Id).
			Update("password", string(hashed)).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "DB update error: " + err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"message": "Password updated successfully",
		})
	})

	app.Post("/signin", func(c *fiber.Ctx) error {
		type Logininput struct {
			Phonenumber string `json:"phone"`
			Password    string `json:"password"`
		}
		var input Logininput
		if err := c.BodyParser(&input); err != nil {
			return c.Status(fiber.ErrBadRequest.Code).JSON(fiber.Map{
				"error": "Invalid Input",
			})
		}
		log.Println("Đang đang nhập: phone=", input.Phonenumber, ", pass= ", input.Password)
		var user *model.User
		var err error
		user, err = signincheck(input.Phonenumber, input.Password)
		if err != nil || user == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		actk, e := handler.GenerateJWT(user.Id)
		if e != nil {
			return c.Status(400).JSON(fiber.Map{"error": "cannot create jwt access token"})
		}
		dvi := c.Get("User-Agent")
		log.Println("Device login:", dvi)

		// log.Println(actk)
		refreshtoken, err := handler.GenerateRefreshToken()
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "cannot create refresh token"})
		}

		var existing model.Token
		result := config.DB.Where("id = ? AND deviceinfo = ?", user.Id, dvi).First(&existing)
		if result.Error == nil {
			// Đã tồn tại → cập nhật lại token thay vì tạo mới
			existing.Accesstoken = actk
			existing.Refreshtoken = refreshtoken
			existing.Created_at = time.Now().Format("2006-01-02 15:04:05")

			if err := config.DB.Save(&existing).Error; err != nil {
				return c.Status(400).JSON(fiber.Map{"error": "cannot update tokens"})
			}
		} else {
			// Chưa có thì tạo mới
			token := model.Token{
				Id:           user.Id,
				Refreshtoken: refreshtoken,
				Accesstoken:  actk,
				Deviceinfo:   dvi,
				Created_at:   time.Now().Format("2006-01-02 15:04:05"),
			}
			if err := config.DB.Create(&token).Error; err != nil {
				return c.Status(400).JSON(fiber.Map{"error": "cannot create tokens"})
			}
		}

		return c.JSON(fiber.Map{
			"message":      "login success",
			"id":           user.Id,
			"refreshtoken": refreshtoken,
			"accesstoken":  actk,
		})

	})
	/// route cho app gửi otp
	app.Get("/getotp", func(c *fiber.Ctx) error {
		var otps []model.OTP

	loop:
		for i := 0; i < 20; i++ {
			select {
			case otp := <-otpQueue:
				otps = append(otps, otp)
			default:
				// Queue rỗng thì break sớm
				break loop
			}
		}

		if len(otps) == 0 {
			return c.Status(204).SendString("No OTP available")
		}

		return c.JSON(otps)
	})
	// app.Patch("/refreshtoken", func(c *fiber.Ctx) error {
	// 	type TokenInput struct {
	// 		Id           uint   `json:"id"`
	// 		Refreshtoken string `json:"refreshtoken"`
	// 		Accesstoken  string `json:"accesstoken"`
	// 	}
	// 	var input TokenInput
	// 	if err := c.BodyParser(&input); err != nil {
	// 		return c.Status(400).JSON(fiber.Map{"error": "Invalid input"})
	// 	}
	// 	token, err := tokencheck(input.Id, input.Refreshtoken, input.Accesstoken)
	// 	if err != nil {
	// 		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
	// 			"error": err.Error(),
	// 		})
	// 	}
	// 	// Trả về token mới
	// 	return c.JSON(fiber.Map{
	// 		"accesstoken": token.Accesstoken,
	// 	})

	// })

	app.Post("/signup", func(c *fiber.Ctx) error {
		var input Signupinput
		if err := c.BodyParser(&input); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid input"})
		}
		if err := signupcheck(&input); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"message": "Signup success"})
	})
	app.Listen(":8081")
}

func genOTP() string {
	max := big.NewInt(1000000) // 6 chữ số
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%06d", n.Int64())
}

type Signupinput struct {
	Phonenumber string `json:"phone"`
	Password    string `json:"password"`
	Name        string `json:"name"`
	Birth       string `json:"birth"`  // Dùng string để dễ dàng xử lý ngày tháng
	Gender      uint   `json:"gender"` //
}

func signupcheck(input *Signupinput) error {
	var user model.User
	result := config.DB.Where("phone = ?", input.Phonenumber).First(&user)
	if result.Error == nil {
		return fmt.Errorf("this phone number is exists")
	}
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return result.Error
	}
	hashpass, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %v", err)
	}
	useradd := model.User{
		Phone:    input.Phonenumber,
		Password: string(hashpass),
		Name:     input.Name,
		Birth:    input.Birth,
		Gender:   input.Gender,
	}
	if err := config.DB.Create(&useradd).Error; err != nil {
		return fmt.Errorf("cannot create information %v", err)
	}
	uid := strconv.FormatUint(uint64(useradd.Id), 10)
	_, err = config.FirestoreClient.Collection("user").Doc(uid).Set(
		context.Background(),
		map[string]interface{}{
			"uid":          uid,
			"phone":        input.Phonenumber,
			"password":     string(hashpass),
			"avatar":       "",
			"background":   "",
			"description":  "",
			"gender":       int(input.Gender),
			"lastseentime": time.Now(),
			"permission":   1, // Mặc định là 1
			"status":       0, // Mặc định là 1 (hoạt động)
			"name":         input.Name,
			"birth":        input.Birth,
		},
	)
	if err != nil {
		return fmt.Errorf("cannot create firestore information %v", err)
	}
	log.Println("User created with phone:", input.Phonenumber)
	return nil
}
func signincheck(inphone, inpass string) (*model.User, error) {
	var user model.User
	result := config.DB.Where("phone = ?", inphone).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("cannot found account")
		}
		return nil, result.Error
	}
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(inpass))
	if err != nil {
		return nil, fmt.Errorf("Invalid password %v", err)
	}
	return &user, nil

}
func tokencheck(inputid uint, inputrefreshtoken, inputaccesstoken string) (*model.Token, error) {
	var token model.Token
	result := config.DB.Where("id = ? ", inputid).Where("refreshtoken = ?", inputrefreshtoken).First(&token)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("cannot find token")
		}
		return nil, result.Error
	}
	newtoken, err := handler.GenerateJWT(inputid)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %v", err)
	}
	err = config.DB.Model(&token).Update("accesstoken", newtoken).Error
	if err != nil {
		return nil, fmt.Errorf("failed update token: %v", err)
	}
	token.Accesstoken = newtoken
	return &token, nil

}
