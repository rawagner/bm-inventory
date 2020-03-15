// Code generated by go-swagger; DO NOT EDIT.

package inventory

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

// PostStepReplyReader is a Reader for the PostStepReply structure.
type PostStepReplyReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *PostStepReplyReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 201:
		result := NewPostStepReplyCreated()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 404:
		result := NewPostStepReplyNotFound()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result

	default:
		return nil, runtime.NewAPIError("unknown error", response, response.Code())
	}
}

// NewPostStepReplyCreated creates a PostStepReplyCreated with default headers values
func NewPostStepReplyCreated() *PostStepReplyCreated {
	return &PostStepReplyCreated{}
}

/*PostStepReplyCreated handles this case with default header values.

Reply accepted
*/
type PostStepReplyCreated struct {
}

func (o *PostStepReplyCreated) Error() string {
	return fmt.Sprintf("[POST /nodes/{node_id}/next-steps/reply][%d] postStepReplyCreated ", 201)
}

func (o *PostStepReplyCreated) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewPostStepReplyNotFound creates a PostStepReplyNotFound with default headers values
func NewPostStepReplyNotFound() *PostStepReplyNotFound {
	return &PostStepReplyNotFound{}
}

/*PostStepReplyNotFound handles this case with default header values.

Node not found
*/
type PostStepReplyNotFound struct {
}

func (o *PostStepReplyNotFound) Error() string {
	return fmt.Sprintf("[POST /nodes/{node_id}/next-steps/reply][%d] postStepReplyNotFound ", 404)
}

func (o *PostStepReplyNotFound) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}