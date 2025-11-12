import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { LoginPage } from './LoginPage';
import { AuthProvider } from '../contexts/AuthContext';
import { LanguageProvider } from '../contexts/LanguageContext';
import { BrowserRouter } from 'react-router-dom';

// Mock the auth context methods
const mockLogin = vi.fn();
const mockVerifyOTP = vi.fn();

vi.mock('../contexts/AuthContext', async () => {
  const actual = await vi.importActual('../contexts/AuthContext');
  return {
    ...actual,
    useAuth: () => ({
      login: mockLogin,
      verifyOTP: mockVerifyOTP,
      user: null,
      loading: false,
    }),
  };
});

// Wrapper component with all providers
const Wrapper = ({ children }: { children: React.ReactNode }) => (
  <BrowserRouter>
    <LanguageProvider>
      <AuthProvider>{children}</AuthProvider>
    </LanguageProvider>
  </BrowserRouter>
);

describe('LoginPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders login form correctly', () => {
    render(<LoginPage />, { wrapper: Wrapper });

    // Check for heading
    expect(screen.getByRole('heading')).toBeInTheDocument();

    // Check for email and password inputs by type
    const emailInputs = screen.getAllByRole('textbox');
    expect(emailInputs.length).toBeGreaterThan(0);

    // Check for submit button
    const buttons = screen.getAllByRole('button');
    expect(buttons.length).toBeGreaterThan(0);
  });

  it('handles form input changes', async () => {
    const { container } = render(<LoginPage />, { wrapper: Wrapper });

    // Find email input by type attribute
    const emailInput = container.querySelector('input[type="email"]') as HTMLInputElement;

    if (emailInput) {
      await userEvent.type(emailInput, 'test@example.com');
      expect(emailInput.value).toBe('test@example.com');
    } else {
      throw new Error('Email input not found');
    }
  });

  it('calls login function when form is submitted', async () => {
    mockLogin.mockResolvedValue({ success: true, requiresOTP: false });

    const { container } = render(<LoginPage />, { wrapper: Wrapper });

    // Find inputs by type attribute
    const emailInput = container.querySelector('input[type="email"]') as HTMLInputElement;
    const passwordInput = container.querySelector('input[type="password"]') as HTMLInputElement;

    if (emailInput && passwordInput) {
      await userEvent.type(emailInput, 'test@example.com');
      await userEvent.type(passwordInput, 'password123');

      // Find and click submit button
      const submitButton = container.querySelector('button[type="submit"]');
      if (submitButton) {
        await userEvent.click(submitButton);

        await waitFor(() => {
          expect(mockLogin).toHaveBeenCalledWith('test@example.com', 'password123');
        });
      }
    }
  });
});
