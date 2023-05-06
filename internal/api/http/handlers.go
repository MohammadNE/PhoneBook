package http

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/MohammadNE/PhoneBook/internal/models"
	"github.com/MohammadNE/PhoneBook/pkg/rdbms"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

func (handler *Server) register(c *fiber.Ctx) error {
	ctx := c.Context()

	request := struct{ Email, Password string }{}
	if err := c.BodyParser(&request); err != nil {
		errString := "Error parsing request body"
		handler.logger.Error(errString, zap.Any("request", request), zap.Error(err))
		return c.Status(http.StatusBadRequest).SendString(errString)
	}

	user, err := handler.repository.GetUserByEmail(ctx, request.Email)
	if err != nil && err.Error() != rdbms.ErrNotFound {
		errString := "Error while retrieving data from database"
		handler.logger.Error(errString, zap.Error(err))
		return c.Status(http.StatusInternalServerError).SendString(errString)
	} else if err == nil || (user != nil && user.Id != 0) {
		errString := "User with given email already exists"
		handler.logger.Error(errString, zap.String("email", request.Email))
		return c.Status(http.StatusBadRequest).SendString(errString)
	}

	user = &models.User{Email: request.Email, Password: request.Password}
	if err := handler.repository.CreateUser(ctx, user); err != nil {
		errString := "Error happened while creating the user"
		handler.logger.Error(errString, zap.Error(err))
		return c.Status(http.StatusInternalServerError).SendString(errString)
	} else if user.Id == 0 {
		errString := "Error invalid user id created"
		handler.logger.Error(errString, zap.Any("user", user))
		return c.Status(http.StatusInternalServerError).SendString(errString)
	}

	token, err := handler.token.CreateTokenString(user.Id)
	if err != nil {
		errString := "Error creating JWT token for user"
		handler.logger.Error(errString, zap.Any("user", user), zap.Error(err))
		return c.Status(http.StatusInternalServerError).SendString(errString)
	}

	response := map[string]string{"Token": token}
	return c.Status(http.StatusCreated).JSON(&response)
}

func (handler *Server) login(c *fiber.Ctx) error {
	ctx := c.Context()

	request := struct{ Email, Password string }{}
	if err := c.BodyParser(&request); err != nil {
		errString := "Error parsing request body"
		handler.logger.Error(errString, zap.Error(err))
		return c.Status(http.StatusBadRequest).SendString(errString)
	}

	user, err := handler.repository.GetUserByEmailAndPassword(ctx, request.Email, request.Password)
	if err != nil {
		errString := "Wrong email or password has been given"
		handler.logger.Error(errString, zap.Error(err))
		return c.Status(http.StatusBadRequest).SendString(errString)
	} else if user == nil {
		errString := "Error invalid user returned"
		handler.logger.Error(errString, zap.Any("request", request))
		return c.Status(http.StatusInternalServerError).SendString(errString)
	}

	token, err := handler.token.CreateTokenString(user.Id)
	if err != nil {
		errString := "Error creating JWT token for user"
		handler.logger.Error(errString, zap.Any("user", user), zap.Error(err))
		return c.Status(http.StatusInternalServerError).SendString(errString)
	}

	response := map[string]string{"Token": token}
	return c.Status(http.StatusOK).JSON(&response)
}

func (handler *Server) getContacts(c *fiber.Ctx) error {
	fmt.Println(c.Locals("user-id"))
	return c.SendStatus(http.StatusNotImplemented)
}

func (handler *Server) createContact(c *fiber.Ctx) error {
	ctx := c.Context()

	userId, ok := c.Locals("user-id").(uint64)
	if !ok || userId == 0 {
		handler.logger.Error("Invalid user-id local")
		return c.SendStatus(http.StatusInternalServerError)
	}

	contact := &models.Contact{}
	if err := c.BodyParser(contact); err != nil {
		errString := "Error parsing request body"
		handler.logger.Error(errString, zap.Any("contact", contact), zap.Error(err))
		return c.Status(http.StatusBadRequest).SendString(errString)
	}
	contact.Id = 0

	if !contact.IsValid() {
		errString := "Error invalid contact has been given"
		handler.logger.Error(errString, zap.Any("contact", contact))
		return c.SendStatus(http.StatusBadRequest)
	}

	if err := handler.repository.CreateContact(ctx, userId, contact); err != nil {
		errString := "Error happened while creating the contact"
		handler.logger.Error(errString, zap.Any("contact", contact), zap.Error(err))
		return c.SendStatus(http.StatusInternalServerError)
	}

	response := "Contact has been created successfully"
	return c.Status(http.StatusCreated).SendString(response)
}

func (handler *Server) getContact(c *fiber.Ctx) error {
	ctx := c.Context()

	userId, ok := c.Locals("user-id").(uint64)
	if !ok || userId == 0 {
		handler.logger.Error("Invalid user-id local")
		return c.SendStatus(http.StatusInternalServerError)
	}

	contactId, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || contactId == 0 {
		handler.logger.Error("Invalid token header", zap.Error(err))
		response := "Invalid contact id in path parameters"
		return c.Status(http.StatusBadRequest).SendString(response)
	}

	contact, err := handler.repository.GetContactById(ctx, userId, contactId)
	if err != nil {
		if err.Error() == rdbms.ErrNotFound {
			response := fmt.Sprintf("The given contact id (%d) doesn't exists", contactId)
			return c.Status(http.StatusBadRequest).SendString(response)
		}

		errString := "Error happened while getting the contact"
		handler.logger.Error(errString, zap.Any("contact", contact), zap.Error(err))
		return c.SendStatus(http.StatusInternalServerError)
	}

	return c.Status(http.StatusOK).JSON(contact)
}

func (handler *Server) updateContact(c *fiber.Ctx) error {
	ctx := c.Context()

	userId, ok := c.Locals("user-id").(uint64)
	if !ok || userId == 0 {
		handler.logger.Error("Invalid user-id local")
		return c.SendStatus(http.StatusInternalServerError)
	}

	contactId, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || contactId == 0 {
		handler.logger.Error("Invalid token header", zap.Error(err))
		response := "Invalid contact id in path parameters"
		return c.Status(http.StatusBadRequest).SendString(response)
	}

	oldContact, err := handler.repository.GetContactById(ctx, userId, contactId)
	if err != nil {
		if err.Error() == rdbms.ErrNotFound {
			response := fmt.Sprintf("The given contact id (%d) doesn't exists", contactId)
			return c.Status(http.StatusBadRequest).SendString(response)
		}

		errString := "Error happened while getting the contact"
		handler.logger.Error(errString, zap.Uint64("user-id", userId), zap.Uint64("contact-id", contactId), zap.Error(err))
		return c.SendStatus(http.StatusInternalServerError)
	}

	newContact := &models.Contact{}
	if err := c.BodyParser(newContact); err != nil {
		errString := "Error parsing request body"
		handler.logger.Error(errString, zap.Any("contact", newContact), zap.Error(err))
		return c.Status(http.StatusBadRequest).SendString(errString)
	}
	newContact.Update(oldContact)

	if err := handler.repository.UpdateContact(ctx, userId, newContact); err != nil {
		errString := "Error happened while creating the contact"
		handler.logger.Error(errString, zap.Any("contact", newContact), zap.Error(err))
		return c.SendStatus(http.StatusInternalServerError)
	}

	response := "Contact has been updated successfully"
	return c.Status(http.StatusOK).SendString(response)
}

func (handler *Server) deleteContact(c *fiber.Ctx) error {
	ctx := c.Context()

	userId, ok := c.Locals("user-id").(uint64)
	if !ok || userId == 0 {
		handler.logger.Error("Invalid user-id local")
		return c.SendStatus(http.StatusInternalServerError)
	}

	contactId, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || contactId == 0 {
		handler.logger.Error("Invalid token header", zap.Error(err))
		response := "Invalid contact id in path parameters"
		return c.Status(http.StatusBadRequest).SendString(response)
	}

	if err := handler.repository.DeleteContact(ctx, userId, contactId); err != nil {
		if err.Error() == rdbms.ErrNotFound {
			response := fmt.Sprintf("The given contact id (%d) doesn't exists", contactId)
			return c.Status(http.StatusBadRequest).SendString(response)
		}

		errString := "Error happened while deleting the contact"
		handler.logger.Error(errString, zap.Uint64("user-id", userId), zap.Uint64("contact-id", contactId), zap.Error(err))
		return c.SendStatus(http.StatusInternalServerError)
	}

	response := "Contact has been deleted successfully"
	return c.Status(http.StatusOK).SendString(response)
}
