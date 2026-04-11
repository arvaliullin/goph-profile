import { createSlice, type PayloadAction } from '@reduxjs/toolkit'
import { clearUserId, saveUserId } from '../../utils/user-id-storage'

export type SessionState = {
  userId: string | null
}

const initialState: SessionState = {
  userId: null,
}

const sessionSlice = createSlice({
  name: 'session',
  initialState,
  reducers: {
    hydrate(state, action: PayloadAction<string | null>) {
      const id = action.payload?.trim() ?? ''
      state.userId = id || null
    },
    login(state, action: PayloadAction<string>) {
      const id = action.payload.trim()
      if (!id) {
        return
      }
      state.userId = id
      saveUserId(id)
    },
    logout(state) {
      state.userId = null
      clearUserId()
    },
  },
})

export const { hydrate, login, logout } = sessionSlice.actions
export default sessionSlice.reducer
