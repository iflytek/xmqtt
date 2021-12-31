package auth

import "xmqtt/utils/log"

type LoginInput struct {
	DeviceName string
	ProductKey string
	ClientId string
	Password string
	ClientAddr string
	Sid string
}

type LoginOutput struct {
	Value string
}

func Login(input LoginInput) (LoginOutput, error){
	output := LoginOutput{}
	log.Infof("login success, input: %+v", input)
	return output, nil
}

type LogoutInput struct {
	Value string
	Sid string
	DeviceName string
	ProductKey string
}

func Logout(input LogoutInput) error {
	log.Infof("logout success, input: %+v", input)
	return nil
}



