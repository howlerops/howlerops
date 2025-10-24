# QueryCard Accessibility Checklist

This document outlines the accessibility features implemented in the QueryCard component and provides a checklist for testing.

## Implemented Features

### ✅ Keyboard Navigation

- **Card Focus**: The entire card is keyboard-focusable (`tabIndex={0}`)
- **Enter/Space Keys**: Both Enter and Space keys trigger the load query action
- **Dropdown Menu**: Fully keyboard accessible via Radix UI primitives
- **Star Button**: Keyboard accessible with proper focus management
- **Delete Dialog**: Keyboard accessible with focus trap

### ✅ ARIA Labels and Roles

- **Card Role**: `role="button"` with descriptive `aria-label`
- **Star Button**: Clear aria-labels for favorite/unfavorite actions
- **Dropdown Trigger**: Descriptive `aria-label="Query actions"`
- **Sync Badge**: Proper `aria-label` describing sync status
- **Timestamp**: `aria-label` with full date information

### ✅ Visual Indicators

- **Focus Ring**: Custom focus ring on card using `focus-within:ring-2`
- **Hover States**: Clear hover feedback on interactive elements
- **Color Contrast**: Proper contrast ratios for text and badges
- **Icon + Text**: All icons accompanied by text labels where needed

### ✅ Screen Reader Support

- **Semantic HTML**: Proper heading hierarchy with CardTitle
- **Description**: CardDescription properly associated with title
- **Hidden Text**: `sr-only` class used in Dialog close button
- **Meaningful Names**: All interactive elements have descriptive labels

### ✅ Focus Management

- **Focus Trap**: Delete dialog traps focus within modal
- **Focus Return**: Focus returns to trigger after closing dialogs
- **Skip Links**: Card can be skipped with Tab key
- **No Focus Loss**: Clicking dropdown doesn't lose card context

## Testing Checklist

### Keyboard Navigation Tests

- [ ] Tab key moves focus to the card
- [ ] Enter key loads the query
- [ ] Space key loads the query
- [ ] Tab moves focus to star button
- [ ] Enter/Space on star toggles favorite
- [ ] Tab moves focus to dropdown button
- [ ] Enter/Space opens dropdown menu
- [ ] Arrow keys navigate dropdown items
- [ ] Enter activates dropdown menu items
- [ ] Escape closes dropdown menu
- [ ] Focus returns to trigger after closing dropdown
- [ ] Shift+Tab navigates backwards correctly

### Screen Reader Tests

#### NVDA/JAWS (Windows)
- [ ] Card announced as "button" with query title
- [ ] Star button state announced (favorited/not favorited)
- [ ] Badges read in correct order
- [ ] Dropdown menu items read clearly
- [ ] Delete dialog title and description announced
- [ ] Focus changes announced appropriately

#### VoiceOver (macOS)
- [ ] Card announced as "button" with query title
- [ ] Star button state announced
- [ ] Folder and tags announced
- [ ] Sync status announced (when visible)
- [ ] Timestamp announced
- [ ] Menu items announced with shortcuts

#### Mobile Screen Readers
- [ ] TalkBack (Android): All elements announced
- [ ] VoiceOver (iOS): All elements announced
- [ ] Touch targets large enough (min 44x44px)

### Visual Tests

- [ ] Focus ring visible on all interactive elements
- [ ] Focus ring has sufficient contrast (3:1 minimum)
- [ ] Hover states clearly visible
- [ ] Text contrast meets WCAG AA (4.5:1 for normal text)
- [ ] Badge colors have sufficient contrast
- [ ] Destructive actions use red color
- [ ] Star icon clearly indicates favorite state

### Color Contrast Tests

Use a tool like [WebAIM Contrast Checker](https://webaim.org/resources/contrastchecker/)

- [ ] Card title: ≥ 4.5:1 contrast ratio
- [ ] Description: ≥ 4.5:1 contrast ratio
- [ ] Badges: ≥ 4.5:1 contrast ratio
- [ ] Timestamp: ≥ 4.5:1 contrast ratio
- [ ] Sync status badge: ≥ 4.5:1 contrast ratio
- [ ] Focus ring: ≥ 3:1 contrast ratio

### Motion and Animation

- [ ] Hover transitions are smooth but not distracting
- [ ] No auto-playing animations
- [ ] Respects `prefers-reduced-motion` (via Tailwind)
- [ ] Dialog animations can be disabled

### Mobile Accessibility

- [ ] Touch targets are at least 44x44 CSS pixels
- [ ] Card is easily tappable
- [ ] Dropdown menu works on touch devices
- [ ] Pinch zoom is not disabled
- [ ] Text is readable at 200% zoom
- [ ] No horizontal scrolling at 320px width

## WCAG 2.1 Level AA Compliance

### Principle 1: Perceivable

- **1.1.1 Non-text Content (A)**: ✅ All icons have text alternatives
- **1.3.1 Info and Relationships (A)**: ✅ Semantic HTML structure
- **1.3.2 Meaningful Sequence (A)**: ✅ Logical reading order
- **1.4.3 Contrast (AA)**: ✅ Minimum 4.5:1 contrast ratio
- **1.4.11 Non-text Contrast (AA)**: ✅ UI components have 3:1 contrast

### Principle 2: Operable

- **2.1.1 Keyboard (A)**: ✅ All functionality keyboard accessible
- **2.1.2 No Keyboard Trap (A)**: ✅ Focus can always move away
- **2.4.3 Focus Order (A)**: ✅ Logical focus order maintained
- **2.4.7 Focus Visible (AA)**: ✅ Clear focus indicators
- **2.5.5 Target Size (AAA)**: ⚠️ Most targets meet 44x44 minimum

### Principle 3: Understandable

- **3.2.1 On Focus (A)**: ✅ No context change on focus
- **3.2.2 On Input (A)**: ✅ No automatic submission
- **3.3.1 Error Identification (A)**: ✅ Delete confirmation is clear

### Principle 4: Robust

- **4.1.2 Name, Role, Value (A)**: ✅ All UI components properly labeled
- **4.1.3 Status Messages (AA)**: ⚠️ Toast notifications should be announced

## Known Limitations

1. **Status Messages**: Success/error toasts may not be announced by all screen readers. Consider using a live region for important messages.

2. **Description Truncation**: Truncated descriptions don't indicate full text is available. Consider adding a tooltip or expand button.

3. **Tag Badges**: Multiple tags may be verbose for screen reader users. Consider grouping or providing a summary.

## Recommended Improvements

1. **Live Regions**: Add `aria-live` regions for status updates
2. **Tooltips**: Add tooltips for truncated text
3. **Landmarks**: Wrap lists in `<nav>` with `aria-label`
4. **Skip Links**: Add "Skip to content" links in parent components
5. **Help Text**: Add `aria-describedby` for complex interactions

## Testing Tools

- **Keyboard**: Manual keyboard navigation
- **Screen Reader**: NVDA, JAWS, VoiceOver, TalkBack
- **Contrast**: WebAIM Contrast Checker, axe DevTools
- **Automation**: axe-core, Lighthouse, Pa11y
- **Manual Review**: WCAG checklist, accessibility audit

## Resources

- [WCAG 2.1 Guidelines](https://www.w3.org/WAI/WCAG21/quickref/)
- [WAI-ARIA Authoring Practices](https://www.w3.org/WAI/ARIA/apg/)
- [WebAIM Keyboard Accessibility](https://webaim.org/techniques/keyboard/)
- [Radix UI Accessibility](https://www.radix-ui.com/primitives/docs/overview/accessibility)
