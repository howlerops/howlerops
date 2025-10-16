import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, userEvent } from '../../utils/test-utils';
import { Button } from '@/components/ui/button';

describe('Button Component', () => {
  let user: ReturnType<typeof userEvent.setup>;

  beforeEach(() => {
    user = userEvent.setup();
  });
  it('renders with default props', () => {
    render(<Button>Click me</Button>);
    const button = screen.getByRole('button', { name: /click me/i });
    expect(button).toBeInTheDocument();
    expect(button).toHaveClass('bg-primary');
  });

  it('renders with different variants', () => {
    const { rerender } = render(<Button variant="default">Default</Button>);
    expect(screen.getByRole('button')).toHaveClass('bg-primary');

    rerender(<Button variant="destructive">Destructive</Button>);
    expect(screen.getByRole('button')).toHaveClass('bg-destructive');

    rerender(<Button variant="outline">Outline</Button>);
    expect(screen.getByRole('button')).toHaveClass('border-input');

    rerender(<Button variant="secondary">Secondary</Button>);
    expect(screen.getByRole('button')).toHaveClass('bg-secondary');

    rerender(<Button variant="ghost">Ghost</Button>);
    expect(screen.getByRole('button')).toHaveClass('hover:bg-accent');

    rerender(<Button variant="link">Link</Button>);
    expect(screen.getByRole('button')).toHaveClass('text-primary');
  });

  it('renders with different sizes', () => {
    const { rerender } = render(<Button size="default">Default</Button>);
    expect(screen.getByRole('button')).toHaveClass('h-10');

    rerender(<Button size="sm">Small</Button>);
    expect(screen.getByRole('button')).toHaveClass('h-9');

    rerender(<Button size="lg">Large</Button>);
    expect(screen.getByRole('button')).toHaveClass('h-11');

    rerender(<Button size="icon">Icon</Button>);
    expect(screen.getByRole('button')).toHaveClass('h-10', 'w-10');
  });

  it('handles click events', async () => {
    const handleClick = vi.fn();

    render(<Button onClick={handleClick}>Click me</Button>);
    const button = screen.getByRole('button');

    await user.click(button);
    expect(handleClick).toHaveBeenCalledTimes(1);
  });

  it('can be disabled', async () => {
    const handleClick = vi.fn();

    render(<Button disabled onClick={handleClick}>Disabled</Button>);
    const button = screen.getByRole('button');

    expect(button).toBeDisabled();
    await user.click(button);
    expect(handleClick).not.toHaveBeenCalled();
  });

  it('supports custom className', () => {
    render(<Button className="custom-class">Custom</Button>);
    const button = screen.getByRole('button');
    expect(button).toHaveClass('custom-class');
  });

  it('forwards ref correctly', () => {
    let buttonRef: HTMLButtonElement | null = null;
    const RefComponent = () => (
      <Button ref={(ref) => { buttonRef = ref; }}>Ref Button</Button>
    );

    render(<RefComponent />);
    expect(buttonRef).toBeInstanceOf(HTMLButtonElement);
    expect(buttonRef?.textContent).toBe('Ref Button');
  });

  it('renders as different element when asChild is true', () => {
    render(
      <Button asChild>
        <a href="/test">Link Button</a>
      </Button>
    );

    const link = screen.getByRole('link');
    expect(link).toBeInTheDocument();
    expect(link).toHaveAttribute('href', '/test');
    expect(link).toHaveClass('bg-primary'); // Should still have button styles
  });

  it('supports keyboard navigation', async () => {
    const handleClick = vi.fn();

    render(<Button onClick={handleClick}>Keyboard Button</Button>);
    const button = screen.getByRole('button');

    button.focus();
    expect(button).toHaveFocus();

    await user.keyboard('{Enter}');
    expect(handleClick).toHaveBeenCalledTimes(1);

    await user.keyboard(' ');
    expect(handleClick).toHaveBeenCalledTimes(2);
  });

  it('has proper accessibility attributes', () => {
    render(<Button aria-label="Accessible button">Icon only</Button>);
    const button = screen.getByRole('button');
    expect(button).toHaveAttribute('aria-label', 'Accessible button');
  });

  it('supports loading state', () => {
    render(<Button disabled>Loading...</Button>);
    const button = screen.getByRole('button');
    expect(button).toBeDisabled();
    expect(button).toHaveTextContent('Loading...');
  });

  describe('Button variants visual regression', () => {
    it('matches snapshot for all variants', () => {
      const variants = ['default', 'destructive', 'outline', 'secondary', 'ghost', 'link'] as const;

      variants.forEach(variant => {
        const { container } = render(<Button variant={variant}>{variant}</Button>);
        expect(container.firstChild).toMatchSnapshot(`button-${variant}`);
      });
    });

    it('matches snapshot for all sizes', () => {
      const sizes = ['default', 'sm', 'lg', 'icon'] as const;

      sizes.forEach(size => {
        const { container } = render(<Button size={size}>{size}</Button>);
        expect(container.firstChild).toMatchSnapshot(`button-${size}`);
      });
    });
  });

  describe('Button interaction states', () => {
    it('shows hover state on mouse enter', async () => {
      render(<Button>Hover me</Button>);
      const button = screen.getByRole('button');

      await user.hover(button);
      expect(button).toHaveClass('hover:bg-primary/90');
    });

    it('shows focus state on keyboard focus', async () => {
      render(<Button>Focus me</Button>);
      const button = screen.getByRole('button');

      button.focus();
      expect(button).toHaveFocus();
      expect(button).toHaveClass('focus-visible:ring-2');
    });

    it('supports tab navigation', async () => {
      render(
        <div>
          <Button>First</Button>
          <Button>Second</Button>
          <Button disabled>Disabled</Button>
          <Button>Third</Button>
        </div>
      );

      const firstButton = screen.getByRole('button', { name: 'First' });
      const secondButton = screen.getByRole('button', { name: 'Second' });
      const thirdButton = screen.getByRole('button', { name: 'Third' });

      firstButton.focus();
      expect(firstButton).toHaveFocus();

      await user.tab();
      expect(secondButton).toHaveFocus();

      await user.tab();
      expect(thirdButton).toHaveFocus(); // Should skip disabled button
    });
  });
});