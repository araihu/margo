package margo

import (
	"fmt"
	"sort"
)

type extensionRegistry struct {
	registrations []ExtensionRegistration
}

// The registry is held behind an unexported field so canonical JSON never
// attempts to encode factories. Identity metadata is stored separately in the
// compiler config and is the only part included in the fingerprint.
type extensionRegistryValue struct {
	registry extensionRegistry
}

func (r extensionRegistry) clone() extensionRegistry {
	out := extensionRegistry{registrations: make([]ExtensionRegistration, len(r.registrations))}
	for i, registration := range r.registrations {
		out.registrations[i] = cloneRegistration(registration)
	}
	return out
}

func cloneRegistration(in ExtensionRegistration) ExtensionRegistration {
	in.Fences = append([]string(nil), in.Fences...)
	in.Identity.Capabilities = append([]string(nil), in.Identity.Capabilities...)
	return in
}

// WithExtension registers one factory before New freezes the registry.
func WithExtension(registration ExtensionRegistration) Option {
	return func(config *compilerConfig) error {
		registry := registryFromConfig(*config)
		if registration.Factory == nil {
			return fmt.Errorf("registry.factory_invalid: extension factory is nil")
		}
		if registration.Identity.Name == "" && registration.Identity.Version == "" && len(registration.Fences) == 0 {
			registration.Identity.Name = fmt.Sprintf("anonymous-%d", len(registry.registrations))
			registration.Identity.Version = "v1"
		}
		if registration.Identity.Name == "" || registration.Identity.Version == "" {
			return fmt.Errorf("registry.identity_invalid: name and version are required")
		}
		if _, err := extensionRegistrationHTMLRequirements(registration); err != nil {
			return err
		}
		for _, existing := range registry.registrations {
			if existing.Identity.Name == registration.Identity.Name {
				return fmt.Errorf("registry.duplicate_name: %s", registration.Identity.Name)
			}
		}
		seenFences := make(map[string]struct{})
		for _, existing := range registry.registrations {
			for _, fence := range existing.Fences {
				seenFences[fence] = struct{}{}
			}
		}
		localFences := make(map[string]struct{}, len(registration.Fences))
		for _, fence := range registration.Fences {
			if fence == "" {
				return fmt.Errorf("registry.fence_invalid: empty fence")
			}
			if _, exists := seenFences[fence]; exists {
				return fmt.Errorf("registry.duplicate_fence: %s", fence)
			}
			if _, exists := localFences[fence]; exists {
				return fmt.Errorf("registry.duplicate_fence: %s", fence)
			}
			localFences[fence] = struct{}{}
		}
		registry.registrations = append(registry.registrations, cloneRegistration(registration))
		config.values["extensionRegistry"] = extensionRegistryValue{registry: registry.clone()}
		config.values["extensionIdentities"] = extensionIdentityPreimage(registry)
		return nil
	}
}

func extensionRegistrationHTMLRequirements(registration ExtensionRegistration) ([]HTMLRequirement, error) {
	requirements := make([]HTMLRequirement, 0, len(registration.Identity.Capabilities))
	byID := make(map[string]HTMLRequirement)
	for _, capability := range registration.Identity.Capabilities {
		requirement, recognized, err := decodeHTMLRequirementCapability(capability)
		if err != nil {
			return nil, err
		}
		if !recognized {
			continue
		}
		if existing, found := byID[requirement.ID]; found {
			if !equalHTMLRequirement(existing, requirement) {
				return nil, htmlRequirementError("html.requirement_conflict", fmt.Sprintf("extension %q declares conflicting requirement %q", registration.Identity.Name, requirement.ID))
			}
			continue
		}
		byID[requirement.ID] = requirement
		requirements = append(requirements, requirement)
	}
	return cloneHTMLRequirements(requirements), nil
}

// WithTheme is the small root theme option consumed by the C4 binding tests;
// the full token/theme registry is owned by later root tasks.
func WithTheme(name string) Option {
	return func(config *compilerConfig) error {
		if name == "" {
			return fmt.Errorf("theme.invalid: theme name is required")
		}
		config.values["theme"] = name
		return nil
	}
}

func registryFromConfig(config compilerConfig) extensionRegistry {
	value, ok := config.values["extensionRegistry"].(extensionRegistryValue)
	if !ok {
		return extensionRegistry{}
	}
	return value.registry.clone()
}

func extensionIdentityPreimage(registry extensionRegistry) []map[string]any {
	registrations := registry.clone().registrations
	sort.SliceStable(registrations, func(i, j int) bool {
		return registrations[i].Identity.Name < registrations[j].Identity.Name
	})
	result := make([]map[string]any, 0, len(registrations))
	for _, registration := range registrations {
		fences := append([]string(nil), registration.Fences...)
		sort.Strings(fences)
		capabilities := append([]string(nil), registration.Identity.Capabilities...)
		sort.Strings(capabilities)
		result = append(result, map[string]any{
			"name":              registration.Identity.Name,
			"version":           registration.Identity.Version,
			"configurationHash": registration.Identity.ConfigurationHash,
			"capabilities":      capabilities,
			"fences":            fences,
		})
	}
	return result
}
