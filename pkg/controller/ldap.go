package controller

import (
	"crypto/tls"
	"errors"
	"fmt"

	"gopkg.in/ldap.v2"
	"gopkg.in/logger.v1"
)

// UserAuthentication support user auth by LDAP account.
func (h *BaseHandler) LdapAuthentication(username, password string) (name string, email string, err error) {
	// The username and password we want to check

	siteConf := h.App.Cf.Site
	l, err := ldap.Dial("tcp", fmt.Sprintf("%s:%d", siteConf.LdapServerHost, siteConf.LdapServerPort))
	if err != nil {
		log.Error(err)
	}
	defer l.Close()

	// Reconnect with TLS
	err = l.StartTLS(&tls.Config{InsecureSkipVerify: true})
	if err != nil {
		log.Error(err)
	}

	// First bind with a read only user
	err = l.Bind(siteConf.LdapBindDN, siteConf.LdapBindPasswd)
	if err != nil {
		log.Error(err)
	}

	// Search for the given username
	searchRequest := ldap.NewSearchRequest(
		siteConf.LdapUserSearchBaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf(siteConf.LdapUserSearchFilter, username),
		[]string{"dn"},
		nil,
	)

	sr, err := l.Search(searchRequest)

	if err != nil {
		log.Error(err)
	}

	if len(sr.Entries) != 1 {
		log.Error("User does not exist or too many entries returned")
	}

	userdn := sr.Entries[0].DN

	// Bind as the user to verify their password
	err = l.Bind(userdn, password)
	if err != nil {
		fmt.Println("userdn: ", userdn, "password: ", password)
		log.Error(err)
	}

	// Rebind as the read only user for any further queries
	err = l.Bind(siteConf.LdapBindDN, siteConf.LdapBindPasswd)
	if err != nil {
		log.Error(err)
	}
	nSearchRequest := ldap.NewSearchRequest(
		"ou=users,dc=changhong,dc=com",
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		// "(cn=*)",
		// fmt.Sprintf("(&(objectClass=organizationalPerson)(uid=%s))", username),
		fmt.Sprintf("(&(objectClass=organizationalPerson)(uid=%s))", username),
		[]string{"uid", "cn", "mail", "userPassword"},
		nil,
	)
	nSr, err := l.Search(nSearchRequest)
	if err != nil {
		log.Error(err)
	}
	fmt.Printf("length:  %d\n", len(nSr.Entries))
	for _, entry := range nSr.Entries {
		email = entry.GetAttributeValue("mail")
		name = entry.GetAttributeValue("uid")
		fmt.Printf("%s: %v\n", entry.DN, entry.GetAttributeValue("mail"))
		// fmt.Printf("%+v\n", entry)
		// for _, v := range entry.Attributes {
		// 	fmt.Printf("%s: %+v\n", v.Name, v.Values)
		// }
		return name, email, nil
	}
	return "", "", errors.New("not verified")
}