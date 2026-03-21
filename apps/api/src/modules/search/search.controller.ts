import { Controller, Get, Query } from "@nestjs/common";
import type { SearchResponse } from "@media-library/types";
import { ProvidersService } from "../providers/providers.service";
import { SearchQueryDto } from "./dto/search-query.dto";

@Controller("search")
export class SearchController {
  constructor(private readonly providersService: ProvidersService) {}

  @Get()
  async search(@Query() query: SearchQueryDto): Promise<SearchResponse> {
    const response = await this.providersService.searchByText({
      mediaType: query.mediaType,
      query: query.q,
      limit: query.limit
    });

    return {
      query: query.q,
      mediaType: query.mediaType,
      results: response.results,
      failures: response.failures.map((failure) => ({
        provider: failure.provider,
        message: failure.message
      }))
    };
  }
}
