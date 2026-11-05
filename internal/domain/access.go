package domain

import "errors"

var (
	ErrInvalidCredentials = errors.New("credenciais inválidas ou acesso não liberado")
	ErrInvalidUser        = errors.New("informe nome, email válido, perfil USER ou ADMIN e senha entre 12 e 72 bytes no cadastro")
	ErrUserNotFound       = errors.New("usuário não encontrado")
	ErrEmailInUse         = errors.New("email já cadastrado")
	ErrLastAdmin          = errors.New("não é possível bloquear ou rebaixar o último administrador ativo")
	ErrSelfAccessChange   = errors.New("não é possível bloquear ou rebaixar a própria conta")
	ErrAdminExists        = errors.New("já existe um administrador; utilize a gestão de usuários")
)
