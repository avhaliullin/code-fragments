package main

import (
	"context"
)

// Наш hello world компонент
type greeter struct{ ctx context.Context }

func NewGreeter(deps greeterDeps) (*greeter, error) {
	return &greeter{ctx: deps.GetContext()}, nil
}

func (g *greeter) Greet() (string, error) {
	return "Hello world!", nil
}

// И его зависимости
type greeterDeps interface {
	GetContext() context.Context
}

// Структура, содержащая ссылки на все компоненты нашего приложения
type app struct {
	ctx context.Context
	g   *greeter
}

// Имплементируем greeterDeps
func (a *app) GetContext() context.Context { return a.ctx }

// Заводим getter для greeter'а - чтобы другие компоненты могли зависеть от него
func (a *app) GetGreeter() *greeter { return a.g }

// initApp инициализирует компоненты нашего приложения
func initApp() (*app, error) {
	appInstance := &app{
		ctx: context.Background(),
	}
	var err error
	appInstance.g, err = NewGreeter(appInstance)
	if err != nil {
		return nil, err
	}
	return appInstance, nil
}

// Положим инстанс приложения в примитивный singleton
var appInstance *app

func getApp() (*app, error) {
	if appInstance == nil {
		return initApp()
	}
	return appInstance, nil
}

// Точка входа в функцию - именно в этот метод будет передано управление
// при вызове нашей функции
func Handler(ctx context.Context, req map[string]string) (interface{}, error) {
	appInstance, err := getApp()
	if err != nil {
		return nil, err
	}

	g := appInstance.GetGreeter()
	return g.Greet()
}
