import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import AuthLayout from '../AuthLayout.vue'

describe('AuthLayout', () => {
  it('delegates to LoginLayout and forwards default and footer slots', () => {
    const wrapper = mount(AuthLayout, {
      slots: {
        default: '<div data-testid="auth-body">Register Form</div>',
        footer: '<div data-testid="auth-footer">Back to Login</div>',
      },
      global: {
        stubs: {
          LoginLayout: {
            template:
              '<section class="gabugu-auth"><div class="body"><slot /></div><div class="footer"><slot name="footer" /></div></section>',
          },
        },
      },
    })

    expect(wrapper.find('.gabugu-auth').exists()).toBe(true)
    expect(wrapper.get('[data-testid="auth-body"]').text()).toBe('Register Form')
    expect(wrapper.get('[data-testid="auth-footer"]').text()).toBe('Back to Login')
  })
})
