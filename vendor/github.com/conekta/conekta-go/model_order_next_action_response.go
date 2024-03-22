/*
Conekta API

Conekta sdk

API version: 2.1.0
Contact: engineering@conekta.com
*/

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

package conekta

import (
	"encoding/json"
)

// checks if the OrderNextActionResponse type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &OrderNextActionResponse{}

// OrderNextActionResponse contains the following attributes that will guide to continue the flow
type OrderNextActionResponse struct {
	RedirectToUrl *OrderNextActionResponseRedirectToUrl `json:"redirect_to_url,omitempty"`
	// Indicates the type of action to be taken
	Type *string `json:"type,omitempty"`
}

// NewOrderNextActionResponse instantiates a new OrderNextActionResponse object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewOrderNextActionResponse() *OrderNextActionResponse {
	this := OrderNextActionResponse{}
	return &this
}

// NewOrderNextActionResponseWithDefaults instantiates a new OrderNextActionResponse object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewOrderNextActionResponseWithDefaults() *OrderNextActionResponse {
	this := OrderNextActionResponse{}
	return &this
}

// GetRedirectToUrl returns the RedirectToUrl field value if set, zero value otherwise.
func (o *OrderNextActionResponse) GetRedirectToUrl() OrderNextActionResponseRedirectToUrl {
	if o == nil || IsNil(o.RedirectToUrl) {
		var ret OrderNextActionResponseRedirectToUrl
		return ret
	}
	return *o.RedirectToUrl
}

// GetRedirectToUrlOk returns a tuple with the RedirectToUrl field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *OrderNextActionResponse) GetRedirectToUrlOk() (*OrderNextActionResponseRedirectToUrl, bool) {
	if o == nil || IsNil(o.RedirectToUrl) {
		return nil, false
	}
	return o.RedirectToUrl, true
}

// HasRedirectToUrl returns a boolean if a field has been set.
func (o *OrderNextActionResponse) HasRedirectToUrl() bool {
	if o != nil && !IsNil(o.RedirectToUrl) {
		return true
	}

	return false
}

// SetRedirectToUrl gets a reference to the given OrderNextActionResponseRedirectToUrl and assigns it to the RedirectToUrl field.
func (o *OrderNextActionResponse) SetRedirectToUrl(v OrderNextActionResponseRedirectToUrl) {
	o.RedirectToUrl = &v
}

// GetType returns the Type field value if set, zero value otherwise.
func (o *OrderNextActionResponse) GetType() string {
	if o == nil || IsNil(o.Type) {
		var ret string
		return ret
	}
	return *o.Type
}

// GetTypeOk returns a tuple with the Type field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *OrderNextActionResponse) GetTypeOk() (*string, bool) {
	if o == nil || IsNil(o.Type) {
		return nil, false
	}
	return o.Type, true
}

// HasType returns a boolean if a field has been set.
func (o *OrderNextActionResponse) HasType() bool {
	if o != nil && !IsNil(o.Type) {
		return true
	}

	return false
}

// SetType gets a reference to the given string and assigns it to the Type field.
func (o *OrderNextActionResponse) SetType(v string) {
	o.Type = &v
}

func (o OrderNextActionResponse) MarshalJSON() ([]byte, error) {
	toSerialize,err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o OrderNextActionResponse) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	if !IsNil(o.RedirectToUrl) {
		toSerialize["redirect_to_url"] = o.RedirectToUrl
	}
	if !IsNil(o.Type) {
		toSerialize["type"] = o.Type
	}
	return toSerialize, nil
}

type NullableOrderNextActionResponse struct {
	value *OrderNextActionResponse
	isSet bool
}

func (v NullableOrderNextActionResponse) Get() *OrderNextActionResponse {
	return v.value
}

func (v *NullableOrderNextActionResponse) Set(val *OrderNextActionResponse) {
	v.value = val
	v.isSet = true
}

func (v NullableOrderNextActionResponse) IsSet() bool {
	return v.isSet
}

func (v *NullableOrderNextActionResponse) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableOrderNextActionResponse(val *OrderNextActionResponse) *NullableOrderNextActionResponse {
	return &NullableOrderNextActionResponse{value: val, isSet: true}
}

func (v NullableOrderNextActionResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableOrderNextActionResponse) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}

