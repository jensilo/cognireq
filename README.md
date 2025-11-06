# CogniReq
The AI-Native Requirements Clarity Platform.
**FOSS**

---

# TODOs

## Backend
- [ ] Implement LLM functionality
- [X] Review rename of `UpdatedIGASession` => `TouchIGASessionUpdatedAt`

## Frontend
- [X] Fix sorting order on questionnaire
  - The current system for managing IGA script question sort_order can cause frontend inconsistencies. When sort_order values are non-contiguous, frontend operations like reordering or displaying UI elements become unreliable due to an implicit assumption of sequential ordering. The backend should normalize sort_order to a contiguous, 0-indexed sequence after create or delete operations to ensure predictable frontend behavior.
- [X] Improve loading state indications (e.g. when chat is sent)
- [ ] Improve error handling within chat/questionnaire
- [ ] Improve sorting UX in questionnaire (have buttons `arrow-up` and `arrow-down` on the left side of a question)