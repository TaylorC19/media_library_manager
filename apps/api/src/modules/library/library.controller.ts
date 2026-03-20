import {
  Body,
  Controller,
  Delete,
  Get,
  Param,
  Patch,
  Post,
  Query
} from "@nestjs/common";
import type {
  DeleteLibraryEntryResponse,
  LibraryEntryResponse,
  ListLibraryEntriesResponse
} from "@media-library/types";
import type { AuthUser } from "@media-library/types";
import { CurrentUser } from "../auth/decorators/current-user.decorator";
import { CreateLibraryEntryDto } from "./dto/create-library-entry.dto";
import { ListLibraryEntriesQueryDto } from "./dto/list-library-entries-query.dto";
import { UpdateLibraryEntryDto } from "./dto/update-library-entry.dto";
import { LibraryService } from "./library.service";

@Controller("library")
export class LibraryController {
  constructor(private readonly libraryService: LibraryService) {}

  @Get()
  listEntries(
    @CurrentUser() user: AuthUser,
    @Query() query: ListLibraryEntriesQueryDto
  ): Promise<ListLibraryEntriesResponse> {
    return this.libraryService.listEntriesForUser(user.id, query);
  }

  @Post()
  createEntry(
    @CurrentUser() user: AuthUser,
    @Body() body: CreateLibraryEntryDto
  ): Promise<LibraryEntryResponse> {
    return this.libraryService.createEntryForUser(user.id, body);
  }

  @Get(":entryId")
  getEntry(
    @CurrentUser() user: AuthUser,
    @Param("entryId") entryId: string
  ): Promise<LibraryEntryResponse> {
    return this.libraryService.getEntryDetailForUser(entryId, user.id);
  }

  @Patch(":entryId")
  updateEntry(
    @CurrentUser() user: AuthUser,
    @Param("entryId") entryId: string,
    @Body() body: UpdateLibraryEntryDto
  ): Promise<LibraryEntryResponse> {
    return this.libraryService.updateEntryForUser(entryId, user.id, body);
  }

  @Delete(":entryId")
  deleteEntry(
    @CurrentUser() user: AuthUser,
    @Param("entryId") entryId: string
  ): Promise<DeleteLibraryEntryResponse> {
    return this.libraryService.deleteEntryForUser(entryId, user.id);
  }
}
