package goonepasslib

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"reflect"
)

type MarshalFunc func(interface{}) ([]byte, error)

func jsonutilMarshalToFile(path string, in interface{}, marshal MarshalFunc) error {
	data, err := marshal(in)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path, data, 0644)
	return err
}

func jsonutilReadFile(path string, out interface{}) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}
	err = json.Unmarshal(content, out)
	if err != nil {
		return err
	}
	return nil
}

func jsonutilWriteFile(path string, in interface{}) error {
	return jsonutilMarshalToFile(path, in, json.Marshal)
}

func WritePrettyFile(path string, in interface{}) error {
	marshalPrettyJson := func(in interface{}) ([]byte, error) {
		data, err := json.MarshalIndent(in, "", "  ")
		return data, err
	}
	return jsonutilMarshalToFile(path, in, marshalPrettyJson)
}



const PlistDocTypeHeader = `<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">`

type PlistXmlElement struct {
	XMLName  xml.Name
	Value    string            `xml:",chardata"`
	Children []PlistXmlElement `xml:",innerxml"`
}

type PlistXml struct {
	XMLName xml.Name `xml:"plist"`
	Root    PlistXmlElement
	Version string `xml:"version,attr"`
}

func tagName(tag string) xml.Name {
	return xml.Name{Local: tag}
}

func plistElement(v interface{}) PlistXmlElement {
	value := reflect.ValueOf(v)
	vType := reflect.TypeOf(v)
	switch vType.Kind() {
	case reflect.Struct:
		elt := PlistXmlElement{XMLName: tagName("dict")}
		for i := 0; i < value.NumField(); i++ {
			field := vType.Field(i)
			if len(field.PkgPath) > 0 {
				// unexported field
				continue
			}

			fieldName := field.Tag.Get("json")
			if fieldName == "" {
				fieldName = field.Name
			}
			elt.Children = append(elt.Children, PlistXmlElement{
				XMLName: tagName("key"),
				Value:   fieldName,
			})
			elt.Children = append(elt.Children, plistElement(value.Field(i).Interface()))
		}
		return elt
	case reflect.Slice:
		if vType.Elem().Kind() == reflect.Uint8 {
			return PlistXmlElement{
				XMLName: tagName("string"),
				Value:   base64.StdEncoding.EncodeToString(value.Interface().([]byte)),
			}
		} else {
			elt := PlistXmlElement{XMLName: tagName("array")}
			for i := 0; i < value.Len(); i++ {
				elt.Children = append(elt.Children, plistElement(value.Index(i).Interface()))
			}
			return elt
		}
	case reflect.Int:
		return PlistXmlElement{
			XMLName: tagName("integer"),
			Value:   fmt.Sprintf("%d", value.Interface().(int)),
		}
	case reflect.String:
		return PlistXmlElement{
			XMLName: tagName("string"),
			Value:   value.String(),
		}
	default:
		panic(fmt.Sprintf("Value type '%s' not supported by PList marshalling", vType.Name()))
	}
}

// Marshal an interface to a PList XML file.
func plistMarshal(v interface{}) (data []byte, err error) {
	defer func() {
		if marshalErr := recover(); marshalErr != nil {
			err = fmt.Errorf("%v", marshalErr)
		}
	}()
	xmlTree := PlistXml{
		Root:    plistElement(v),
		Version: "1.0",
	}
	data, err = xml.MarshalIndent(xmlTree, "", "\t")
	if err == nil {
		data = []byte(fmt.Sprintf("%s%s\n%s", xml.Header, PlistDocTypeHeader, data))
	}
	return
}
