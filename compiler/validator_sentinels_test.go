package compiler_test

import (
	"errors"
	"testing"

	"github.com/GabrielNunesIT/gotya/compiler"
)

// TestValidatorSentinels_IdentityrefBase verifies that an invalid identityref base
// emits ErrIdentityrefBase (not ErrInvalidDefault).
func TestValidatorSentinels_IdentityrefBase(t *testing.T) {
	t.Parallel()

	input := `
		module sentinel-test {
			namespace "urn:sentinel-test";
			prefix "st";

			leaf iref-leaf {
				type identityref {
					base nonexistent-identity;
				}
			}
		}
	`
	_, err := compile(t, input)
	if err == nil {
		t.Fatal("expected compilation error for invalid identityref base, got nil")
	}
	if !errors.Is(err, compiler.ErrIdentityrefBase) {
		t.Errorf("errors.Is(err, ErrIdentityrefBase) = false; err = %v", err)
	}
}

// TestValidatorSentinels_InvalidDefault verifies that an invalid default value
// emits ErrInvalidDefault.
func TestValidatorSentinels_InvalidDefault(t *testing.T) {
	t.Parallel()

	input := `
		module sentinel-test2 {
			namespace "urn:sentinel-test2";
			prefix "st2";

			leaf flag {
				type boolean;
				default "notabool";
			}
		}
	`
	_, err := compile(t, input)
	if err == nil {
		t.Fatal("expected compilation error for invalid default value, got nil")
	}
	if !errors.Is(err, compiler.ErrInvalidDefault) {
		t.Errorf("errors.Is(err, ErrInvalidDefault) = false; err = %v", err)
	}
}

// TestValidatorSentinels_XPathSyntax verifies that a malformed XPath expression
// emits ErrXPathSyntax.
func TestValidatorSentinels_XPathSyntax(t *testing.T) {
	t.Parallel()

	input := `
		module sentinel-test3 {
			namespace "urn:sentinel-test3";
			prefix "st3";

			leaf flagx {
				type string;
				when "mismatched(";
			}
		}
	`
	_, err := compile(t, input)
	if err == nil {
		t.Fatal("expected compilation error for malformed XPath, got nil")
	}
	if !errors.Is(err, compiler.ErrXPathSyntax) {
		t.Errorf("errors.Is(err, ErrXPathSyntax) = false; err = %v", err)
	}
}
