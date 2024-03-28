// Code generated by templ - DO NOT EDIT.

// templ: version: v0.2.648
package templates

//lint:file-ignore SA4006 This context is only used if a nested component is present.

import "github.com/a-h/templ"
import "context"
import "io"
import "bytes"

func GetOnboardingModal() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, templ_7745c5c3_W io.Writer) (templ_7745c5c3_Err error) {
		templ_7745c5c3_Buffer, templ_7745c5c3_IsBuffer := templ_7745c5c3_W.(*bytes.Buffer)
		if !templ_7745c5c3_IsBuffer {
			templ_7745c5c3_Buffer = templ.GetBuffer()
			defer templ.ReleaseBuffer(templ_7745c5c3_Buffer)
		}
		ctx = templ.InitializeContext(ctx)
		templ_7745c5c3_Var1 := templ.GetChildren(ctx)
		if templ_7745c5c3_Var1 == nil {
			templ_7745c5c3_Var1 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<div id=\"modal\" _=\"on closeModal add .closing then wait for animationend then remove me\"><div class=\"modal-underlay\"></div><form class=\"modal-content\" hx-post=\"/admin/post/config\" hx-target=\"#listener-address\"><h2>Set Up Drawbridge</h2><label for=\"listener-address\">What IP should Drawbridge listen on?</label><p class=\"note-text\">Note: this is the address your Emissary clients will use to connect to Drawbridge. It can be a LAN or WAN address.</p><input name=\"listener-address\" type=\"text\" id=\"listener-address\" placeholder=\"50.42.165.84\"> <label for=\"enable-ping\"><input type=\"checkbox\" id=\"enable-ping\" name=\"enable-ping\" checked> Automatically send anonymous and private daily usage ping to Dawson to estimate acive users. </label><p class=\"note-text\">Note: this feature does not collect your IP address.</p><input name=\"submit-config\" type=\"submit\" id=\"submit-config\" _=\"on click trigger closeModal\"></form></div>")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		if !templ_7745c5c3_IsBuffer {
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteTo(templ_7745c5c3_W)
		}
		return templ_7745c5c3_Err
	})
}
